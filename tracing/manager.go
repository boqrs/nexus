package tracing

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"google.golang.org/grpc/credentials"
)

// ProviderManager 管理 TracerProvider 的单例和热更新
type ProviderManager struct {
	mu             sync.RWMutex
	currentTP      *sdktrace.TracerProvider
	dynamicSampler *DynamicSampler
	config         *Config
	shutdownFunc   func(context.Context) error
}

var (
	globalManager *ProviderManager
	initOnce      sync.Once
)

// InitGlobalTracer 初始化全局链路追踪。必须在应用启动时调用一次。
func InitGlobalTracer(cfg *Config) (*ProviderManager, error) {
	var err error
	initOnce.Do(func() {
		globalManager, err = newProviderManager(cfg)
	})
	return globalManager, err
}

// GetManager 获取全局 Manager 实例
func GetManager() *ProviderManager {
	return globalManager
}

func newProviderManager(cfg *Config) (*ProviderManager, error) {
	// 1. 创建动态采样器
	ds := NewDynamicSampler(cfg.SampleRate)

	// 2. 创建初始 Provider
	tp, shutdown, err := createTracerProvider(cfg, ds)
	if err != nil {
		return nil, err
	}

	// 3. 设置全局 Provider
	otel.SetTracerProvider(tp)

	pm := &ProviderManager{
		currentTP:      tp,
		dynamicSampler: ds,
		config:         cfg,
		shutdownFunc:   shutdown,
	}

	log.Printf("Tracing: Initialized successfully. Service=%s, SampleRate=%.2f", cfg.ServiceName, cfg.SampleRate)
	return pm, nil
}

// UpdateConfig 热更新配置
func (pm *ProviderManager) UpdateConfig(newCfg *Config) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 场景 A: 仅采样率变化 (轻量级，无锁争用，原子更新)
	if newCfg.SampleRate != pm.config.SampleRate {
		pm.dynamicSampler.SetRate(newCfg.SampleRate)
		pm.config.SampleRate = newCfg.SampleRate
		log.Printf("Tracing: Sample rate updated to %.2f", newCfg.SampleRate)
	}

	// 场景 B: Exporter 端点、类型或 Headers 变化 (重量级，需重建 Provider)
	// 注意：如果 Headers 变了（比如 Token 刷新），也需要重建 Exporter
	if newCfg.Exporter.Endpoint != pm.config.Exporter.Endpoint ||
		newCfg.Exporter.Type != pm.config.Exporter.Type {

		log.Println("Tracing: Exporter configuration changed. Rebuilding Provider...")

		// 1. 创建新的 Provider
		newTP, newShutdown, err := createTracerProvider(newCfg, pm.dynamicSampler)
		if err != nil {
			return fmt.Errorf("failed to create new tracer provider: %w", err)
		}

		// 2. 保存旧引用
		oldTP := pm.currentTP
		oldShutdown := pm.shutdownFunc

		// 3. 原子替换全局 Provider
		otel.SetTracerProvider(newTP)

		// 4. 更新内部状态
		pm.currentTP = newTP
		pm.shutdownFunc = newShutdown
		pm.config = newCfg

		// 5. 异步关闭旧 Provider (防止阻塞配置更新流程)
		go func(oldTP *sdktrace.TracerProvider) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := oldShutdown(ctx); err != nil {
				log.Printf("Tracing: Error shutting down old provider: %v", err)
			} else {
				log.Println("Tracing: Old provider shut down successfully")
			}
		}(oldTP)
	}

	return nil
}

// Shutdown 优雅关闭全局 TracerProvider
func (pm *ProviderManager) Shutdown(ctx context.Context) error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	if pm.shutdownFunc != nil {
		return pm.shutdownFunc(ctx)
	}
	return nil
}

// createTracerProvider 内部辅助函数：创建 OTel Provider
func createTracerProvider(cfg *Config, sampler *DynamicSampler) (*sdktrace.TracerProvider, func(context.Context) error, error) {
	if !cfg.Enabled {
		// 如果禁用，返回一个空的 Provider 或者报错，这里简单处理为不创建
		// 实际生产中可能需要一个 NoOp Provider，但 OTel 默认全局就是 NoOp 如果没设置
		return nil, func(context.Context) error { return nil }, nil
	}

	// 1. 创建 Exporter (目前仅支持 OTLP gRPC)
	var exporter sdktrace.SpanExporter
	var err error

	if cfg.Exporter.Type == "otlp_grpc" {
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(cfg.Exporter.Endpoint),
			otlptracegrpc.WithTimeout(cfg.Exporter.Timeout),
		}

		// 处理 TLS/Insecure
		if cfg.Exporter.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		} else {
			opts = append(opts, otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")))
		}

		// 【关键修改】处理鉴权 Headers (例如阿里云 ARMS)
		// 如果配置了 Headers，将其传递给 gRPC Exporter
		if len(cfg.Exporter.Authorization) > 0 {
			header := make(map[string]string)
			header["authentication"] = cfg.Exporter.Authorization
			opts = append(opts, otlptracegrpc.WithHeaders(header))
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		exporter, err = otlptracegrpc.New(ctx, opts...)
	} else {
		return nil, nil, fmt.Errorf("unsupported exporter type: %s", cfg.Exporter.Type)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// 2. 创建 Resource
	attrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(cfg.ServiceName),
		semconv.ServiceVersionKey.String(cfg.ServiceVersion),
	}
	for k, v := range cfg.ResourceAttributes {
		attrs = append(attrs, attribute.String(k, v))
	}

	res, err := resource.New(context.Background(), resource.WithAttributes(attrs...))
	if err != nil {
		return nil, nil, err
	}

	// 3. 创建 TracerProvider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		// 使用 ParentBased 包装动态采样器，尊重上游采样决策
		sdktrace.WithSampler(sdktrace.ParentBased(sampler)),
	)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader)),
	))

	return tp, tp.Shutdown, nil
}

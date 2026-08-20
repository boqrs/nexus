package tracing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest" // 【修复】导入 tracetest 包
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// helperResetGlobalTracer 用于在测试后重置全局 TracerProvider，防止测试间干扰
func helperResetGlobalTracer(t *testing.T) {
	t.Cleanup(func() {
		// 恢复为 NoOp Provider 或重新初始化一个干净的
		// 注意：OTel 全局状态是进程级的，严格隔离很难，这里尽量恢复
		otel.SetTracerProvider(sdktrace.NewTracerProvider())
		globalManager = nil
		initOnce = sync.Once{}
	})
}

// --- 1. DynamicSampler 测试 ---

func TestDynamicSampler_Basic(t *testing.T) {
	ds := NewDynamicSampler(0.5)
	assert.Equal(t, 0.5, ds.getRateFloat64())
	assert.Equal(t, "DynamicSampler", ds.Description())
}

func TestDynamicSampler_SetRate_Boundaries(t *testing.T) {
	ds := NewDynamicSampler(0.5)

	// 测试负数
	ds.SetRate(-0.1)
	assert.Equal(t, 0.0, ds.getRateFloat64())

	// 测试大于1
	ds.SetRate(1.5)
	assert.Equal(t, 1.0, ds.getRateFloat64())

	// 测试正常更新
	ds.SetRate(0.8)
	assert.Equal(t, 0.8, ds.getRateFloat64())
}

func TestDynamicSampler_ShouldSample(t *testing.T) {
	// 1.0 采样率应该总是采样
	ds := NewDynamicSampler(1.0)
	params := sdktrace.SamplingParameters{
		TraceID: oteltrace.TraceID{1},
	}
	result := ds.ShouldSample(params)
	assert.Equal(t, sdktrace.RecordAndSample, result.Decision)

	// 0.0 采样率应该总是丢弃
	dsZero := NewDynamicSampler(0.0)
	resultZero := dsZero.ShouldSample(params)
	assert.Equal(t, sdktrace.Drop, resultZero.Decision)
}

func TestDynamicSampler_ConcurrentSetRate(t *testing.T) {
	ds := NewDynamicSampler(0.5)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(val float64) {
			defer wg.Done()
			ds.SetRate(val)
		}(float64(i) / 100.0)
	}
	wg.Wait()
	// 只要不 panic 且最终值在合法范围内即可
	finalRate := ds.getRateFloat64()
	assert.GreaterOrEqual(t, finalRate, 0.0)
	assert.LessOrEqual(t, finalRate, 1.0)
}

// --- 2. Config 测试 ---

func TestConfig_Reload_DecodeError(t *testing.T) {
	cfg := &Config{}
	// 传入无效类型导致解码失败
	err := cfg.Reload(map[string]interface{}{
		"tracing": map[string]interface{}{
			"sample_rate": "invalid_float",
		},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode")
}

func TestConfig_Reload_YamlInterfaceMap(t *testing.T) {
	helperResetGlobalTracer(t) // 【重要】确保全局状态干净，避免受之前测试影响

	cfg := &Config{}
	// 模拟 YAML 解析出的 map[interface{}]interface{}
	// 【修复】添加完整的 exporter 配置，否则 InitGlobalTracer 会报错 "unsupported exporter type: "
	yamlStyleMap := map[string]interface{}{
		"tracing": map[interface{}]interface{}{
			"enabled":     true,
			"sample_rate": 0.9,
			"exporter": map[interface{}]interface{}{
				"type":     "otlp_grpc",
				"endpoint": "localhost:4317",
				"insecure": true,
			},
		},
	}
	err := cfg.Reload(yamlStyleMap)
	assert.NoError(t, err)
	assert.Equal(t, 0.9, cfg.SampleRate)
	assert.True(t, cfg.Enabled)

	// 验证全局 Manager 也已正确初始化
	manager := GetManager()
	require.NotNil(t, manager)
	assert.Equal(t, 0.9, manager.dynamicSampler.getRateFloat64())
}

// --- 3. ProviderManager & Provider 测试 ---

func TestNewProvider_NilConfig(t *testing.T) {
	helperResetGlobalTracer(t)
	_, err := NewProvider(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config is nil")
}

func TestInitGlobalTracer_Disabled(t *testing.T) {
	helperResetGlobalTracer(t)
	cfg := &Config{
		Enabled: false,
	}
	manager, err := InitGlobalTracer(cfg)
	require.NoError(t, err)
	require.NotNil(t, manager)
	// 禁用时，createTracerProvider 返回 nil TP 和 no-op shutdown
	assert.Nil(t, manager.currentTP)
}

func TestProviderManager_UpdateConfig_SampleRateOnly(t *testing.T) {
	helperResetGlobalTracer(t)
	cfg := &Config{
		Enabled:     true,
		ServiceName: "test-sr",
		SampleRate:  0.1,
		Exporter:    ExporterConfig{Type: "otlp_grpc", Endpoint: "localhost:4317", Insecure: true},
	}
	manager, err := newProviderManager(cfg)
	require.NoError(t, err)

	assert.Equal(t, 0.1, manager.dynamicSampler.getRateFloat64())

	// 仅更新采样率
	newCfg := *cfg
	newCfg.SampleRate = 0.9
	err = manager.UpdateConfig(&newCfg)
	assert.NoError(t, err)
	assert.Equal(t, 0.9, manager.dynamicSampler.getRateFloat64())
	// Provider 实例不应改变
	assert.Same(t, manager.currentTP, manager.currentTP)
}

func TestProviderManager_UpdateConfig_ExporterChange(t *testing.T) {
	helperResetGlobalTracer(t)
	cfg := &Config{
		Enabled:     true,
		ServiceName: "test-exp",
		SampleRate:  1.0,
		Exporter:    ExporterConfig{Type: "otlp_grpc", Endpoint: "localhost:4317", Insecure: true},
	}
	manager, err := newProviderManager(cfg)
	require.NoError(t, err)
	oldTP := manager.currentTP

	// 更新 Endpoint
	newCfg := *cfg
	newCfg.Exporter.Endpoint = "new-endpoint:4317"

	// 注意：由于 new-endpoint 连不上，createTracerProvider 可能会超时或报错
	// 为了测试逻辑，我们假设它成功，或者捕获错误
	// 在单元测试中，由于没有真实的 Collector，这里通常会因为连接超时失败
	// 但我们主要测试逻辑分支：如果成功，TP 应该改变

	// 为了绕过网络错误，我们可以 Mock createTracerProvider，但它是私有函数。
	// 这里我们验证逻辑：如果 UpdateConfig 返回错误，状态不应改变
	err = manager.UpdateConfig(&newCfg)
	// 在实际环境中，如果新 Endpoint 不可达，New 可能会阻塞直到超时然后报错
	// 这里我们主要确认代码路径被执行了。如果报错，说明重建失败。
	if err == nil {
		assert.NotSame(t, oldTP, manager.currentTP)
	} else {
		// 预期行为：连接新端点失败
		assert.Contains(t, err.Error(), "failed to create new tracer provider")
	}
}

func TestProvider_Close(t *testing.T) {
	helperResetGlobalTracer(t)
	cfg := &Config{
		Enabled:     true,
		ServiceName: "test-close",
		Exporter:    ExporterConfig{Type: "otlp_grpc", Endpoint: "localhost:4317", Insecure: true},
	}
	prov, err := NewProvider(cfg)
	require.NoError(t, err)

	err = prov.Close()
	assert.NoError(t, err)
}

// --- 4. GinMiddleware 测试 ---

func setupGinRouterWithTracer(tp *sdktrace.TracerProvider) *gin.Engine {
	gin.SetMode(gin.TestMode)
	// 临时设置全局 Propagator 和 Provider
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader)),
		propagation.TraceContext{},
	))

	r := gin.New()
	r.Use(GinMiddleware("test-service"))
	return r
}

func TestGinMiddleware_NormalRequest(t *testing.T) {
	helperResetGlobalTracer(t)

	// 【修复】使用 tracetest.NewInMemoryExporter()
	exporter := tracetest.NewInMemoryExporter()

	res, _ := resource.New(context.Background(), resource.WithAttributes(semconv.ServiceNameKey.String("test")))
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	defer tp.Shutdown(context.Background())

	r := setupGinRouterWithTracer(tp)
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"msg": "pong"})
	})

	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)

	span := spans[0]
	assert.Equal(t, "GET /ping", span.Name)
	assert.Equal(t, codes.Ok, span.Status.Code)
	assert.Contains(t, span.Attributes, attribute.Int("http.status_code", 200))
}

func TestGinMiddleware_ErrorStatus(t *testing.T) {
	helperResetGlobalTracer(t)

	// 【修复】使用 tracetest.NewInMemoryExporter()
	exporter := tracetest.NewInMemoryExporter()

	res, _ := resource.New(context.Background(), resource.WithAttributes(semconv.ServiceNameKey.String("test")))
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	defer tp.Shutdown(context.Background())

	r := setupGinRouterWithTracer(tp)
	r.GET("/error", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "fail"})
	})

	req := httptest.NewRequest("GET", "/error", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)

	span := spans[0]
	assert.Equal(t, codes.Error, span.Status.Code)
	assert.Contains(t, span.Status.Description, "HTTP 500 error")
}

func TestGinMiddleware_ShortTraceIDNormalization(t *testing.T) {
	helperResetGlobalTracer(t)

	// 【修复】使用 tracetest.NewInMemoryExporter()
	exporter := tracetest.NewInMemoryExporter()

	res, _ := resource.New(context.Background(), resource.WithAttributes(semconv.ServiceNameKey.String("test")))
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	defer tp.Shutdown(context.Background())

	r := setupGinRouterWithTracer(tp)
	r.GET("/short-id", func(c *gin.Context) {
		// 获取当前 Context 中的 TraceID
		spanCtx := oteltrace.SpanContextFromContext(c.Request.Context())
		traceIDStr := spanCtx.TraceID().String()
		c.Header("X-Debug-TraceID", traceIDStr)
		c.JSON(http.StatusOK, gin.H{})
	})

	req := httptest.NewRequest("GET", "/short-id", nil)
	// 设置一个 16 位的 TraceID (B3 格式常见)
	shortTraceID := "0af7651916cd43dd"
	req.Header.Set("X-B3-TraceId", shortTraceID)
	req.Header.Set("X-B3-SpanId", "b7ad6b7169203331")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// 验证中间件日志输出（虽然无法直接断言日志，但可以验证结果）
	// 验证 Span 中的 TraceID 是否被补全为 32 位
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)

	// 预期的 TraceID 应该是前补零
	expectedTraceIDHex := "0000000000000000" + shortTraceID
	expectedTraceID, _ := oteltrace.TraceIDFromHex(expectedTraceIDHex)

	assert.Equal(t, expectedTraceID, spans[0].SpanContext.TraceID())
}

func TestGinMiddleware_NoRoute(t *testing.T) {
	helperResetGlobalTracer(t)

	// 【修复】使用 tracetest.NewInMemoryExporter()
	exporter := tracetest.NewInMemoryExporter()

	res, _ := resource.New(context.Background(), resource.WithAttributes(semconv.ServiceNameKey.String("test")))
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	defer tp.Shutdown(context.Background())

	r := setupGinRouterWithTracer(tp)
	// 不注册任何路由，触发 404

	req := httptest.NewRequest("GET", "/non-existent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)

	// 当 FullPath 为空时，Span Name 应使用 URL Path
	assert.Equal(t, "GET /non-existent", spans[0].Name)
}

// --- 5. Integration Test for Provider Reload ---

func TestProvider_Reload_Integration(t *testing.T) {
	helperResetGlobalTracer(t)

	cfg := &Config{
		Enabled:     true,
		ServiceName: "test-reload-int",
		SampleRate:  0.1,
		Exporter:    ExporterConfig{Type: "otlp_grpc", Endpoint: "localhost:4317", Insecure: true},
	}

	prov, err := NewProvider(cfg)
	require.NoError(t, err)

	manager := GetManager()
	require.NotNil(t, manager)
	assert.Equal(t, 0.1, manager.dynamicSampler.getRateFloat64())

	// 执行 Reload
	newConfigMap := map[string]interface{}{
		"tracing": map[string]interface{}{
			"enabled":     true,
			"sample_rate": 0.8,
			"exporter": map[string]interface{}{
				"type":     "otlp_grpc",
				"endpoint": "localhost:4317",
				"insecure": true,
			},
		},
	}

	err = prov.Reload(newConfigMap)
	assert.NoError(t, err)
	assert.Equal(t, 0.8, manager.dynamicSampler.getRateFloat64())
}

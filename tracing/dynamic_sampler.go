package tracing

import (
	"math"
	"sync/atomic"

	"go.opentelemetry.io/otel/sdk/trace"
)

// DynamicSampler 是一个支持动态修改采样率的 Sampler
type DynamicSampler struct {
	rateBits atomic.Uint64
	// 使用 atomic.Pointer 来原子地替换 delegate，避免 Write-Write Race
	delegate atomic.Pointer[trace.Sampler]
}

// NewDynamicSampler 创建动态采样器
func NewDynamicSampler(initialRate float64) *DynamicSampler {
	ds := &DynamicSampler{}
	ds.setRateFloat64(initialRate)
	return ds
}

func (ds *DynamicSampler) setRateFloat64(rate float64) {
	ds.rateBits.Store(math.Float64bits(rate))
	// 创建新的 sampler 实例
	newDelegate := trace.TraceIDRatioBased(rate)
	// 原子地存储指针
	ds.delegate.Store(&newDelegate)
}

func (ds *DynamicSampler) getRateFloat64() float64 {
	bits := ds.rateBits.Load()
	return math.Float64frombits(bits)
}

// SetRate 动态更新采样率
func (ds *DynamicSampler) SetRate(rate float64) {
	if rate < 0 {
		rate = 0
	}
	if rate > 1 {
		rate = 1
	}

	currentRate := ds.getRateFloat64()
	if currentRate != rate {
		ds.setRateFloat64(rate)
	}
}

// ShouldSample 实现 trace.Sampler 接口
func (ds *DynamicSampler) ShouldSample(parameters trace.SamplingParameters) trace.SamplingResult {
	// 原子地加载当前 delegate 指针
	// Load() 返回的是 *trace.Sampler，所以需要解引用
	sampler := ds.delegate.Load()
	if sampler == nil {
		// 防御性编程：理论上不会发生，因为 NewDynamicSampler 会初始化
		return trace.SamplingResult{Decision: trace.Drop}
	}
	return (*sampler).ShouldSample(parameters)
}

// Description 实现 trace.Sampler 接口
func (ds *DynamicSampler) Description() string {
	return "DynamicSampler"
}

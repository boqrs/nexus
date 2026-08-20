package metrics

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/push"
)

var (
	// defaultProvider is the default metric provider.
	defaultProvider = NewProvider(&Config{})
)

// Provider is a metric provider that is responsible for creating and managing metrics.
type Provider struct {
	mu       sync.RWMutex
	registry *prometheus.Registry
	pusher   *push.Pusher
	cancel   context.CancelFunc
	cfg      *Config
}

// NewProvider creates a new metric provider.
func NewProvider(cfg *Config) *Provider {
	p := &Provider{}
	p.ApplyConfig(cfg)
	return p
}

// ApplyConfig applies a new configuration to the provider.
// This is the core of the hot-reloading mechanism.
func (p *Provider) ApplyConfig(cfg *Config) {
	p.mu.Lock()
	defer p.mu.Unlock()

	cfg.SetDefault()
	p.cfg = cfg

	// Create a new registry
	p.registry = prometheus.NewRegistry()
	p.registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	p.registry.MustRegister(prometheus.NewGoCollector())

	// If there's a previous pusher, stop it.
	if p.cancel != nil {
		p.cancel()
	}

	// If push gateway is enabled, create a new pusher.
	if cfg.PushGateway.Enabled && cfg.PushGateway.URL != "" {
		p.pusher = push.New(cfg.PushGateway.URL, cfg.PushGateway.Job).Gatherer(p.registry)
		if len(p.cfg.ConstLabels) > 0 {
			p.pusher.Grouping("instance", p.cfg.ConstLabels["instance"])
		}

		var ctx context.Context
		ctx, p.cancel = context.WithCancel(context.Background())

		go func(ctx context.Context) {
			// TODO: Replace this with a proper ticker implementation
			for {
				select {
				case <-ctx.Done():
					return
				default:
					if err := p.pusher.Push(); err != nil {
						fmt.Printf("could not push metrics to gateway: %v", err)
					}
				}
			}
		}(ctx)
	}
}

// NewCounter creates a new counter metric.
func NewCounter(opts prometheus.CounterOpts) prometheus.Counter {
	return defaultProvider.NewCounter(opts)
}

// NewGauge creates a new gauge metric.
func NewGauge(opts prometheus.GaugeOpts) prometheus.Gauge {
	return defaultProvider.NewGauge(opts)
}

// NewHistogram creates a new histogram metric.
func NewHistogram(opts prometheus.HistogramOpts) prometheus.Histogram {
	return defaultProvider.NewHistogram(opts)
}

// Handler returns the http.Handler for the default provider.
func Handler() http.Handler {
	return defaultProvider.Handler()
}

// NewCounter creates a new counter metric for this provider.
func (p *Provider) NewCounter(opts prometheus.CounterOpts) prometheus.Counter {
	return promauto.With(p.registry).NewCounter(opts)
}

// NewGauge creates a new gauge metric for this provider.
func (p *Provider) NewGauge(opts prometheus.GaugeOpts) prometheus.Gauge {
	return promauto.With(p.registry).NewGauge(opts)
}

// NewHistogram creates a new histogram metric for this provider.
func (p *Provider) NewHistogram(opts prometheus.HistogramOpts) prometheus.Histogram {
	return promauto.With(p.registry).NewHistogram(opts)
}

// Handler returns an http.Handler for this provider. It is a wrapper that
// ensures the current registry is used, making it safe for hot-reloading.
func (p *Provider) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p.mu.RLock()
		defer p.mu.RUnlock()
		h := promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	})
}

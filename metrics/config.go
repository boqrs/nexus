package metrics

import "time"

// Config is the configuration for the metrics provider.
type Config struct {
	Namespace   string            `json:"namespace" yaml:"namespace" mapstructure:"namespace"`
	ConstLabels map[string]string `json:"const_labels" yaml:"const_labels" mapstructure:"const_labels"`
	PushGateway PushGatewayConfig `json:"push_gateway" yaml:"push_gateway" mapstructure:"push_gateway"`
}

// PushGatewayConfig is the configuration for the Prometheus Pushgateway.
type PushGatewayConfig struct {
	Enabled  bool          `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	URL      string        `json:"url" yaml:"url" mapstructure:"url"`
	Job      string        `json:"job" yaml:"job" mapstructure:"job"`
	Interval time.Duration `json:"interval" yaml:"interval" mapstructure:"interval"`
}

// SetDefault sets the default values for the configuration.
func (c *Config) SetDefault() {
	if c.PushGateway.Interval == 0 {
		c.PushGateway.Interval = 15 * time.Second
	}
	if c.PushGateway.Job == "" {
		c.PushGateway.Job = "comm"
	}
}

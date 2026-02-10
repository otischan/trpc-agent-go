package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

// Config represents the application configuration
type Config struct {
	LogLevel     string        `yaml:"log_level"`
	MetricsPort  int           `yaml:"metrics_port"`
	Namespace    string        `yaml:"namespace"`
	Kubeconfig   string        `yaml:"kubeconfig"`
	LLM          LLM           `yaml:"llm"`
	MCP          MCP           `yaml:"mcp"`
	Rules        Rules         `yaml:"rules"`
	Basic        Basic         `yaml:"basic"`
	Monitoring   Monitoring    `yaml:"monitoring"`
	Monitor      MonitorConfig `yaml:"monitor"` // Monitor module configuration
	RuleEngine   RuleEngine    `yaml:"rule_engine"`
	AIDecision   AIDecision    `yaml:"ai_decision"`
	K8sOperation K8sOperation  `yaml:"k8s_operation"`
}

// LLM holds LLM-specific configurations
type LLM struct {
	Model   string `yaml:"model"`
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
	Enabled bool   `yaml:"enabled"`
	Variant string `yaml:"variant"`
}

// MCPServer holds individual MCP server configuration
type MCPServer struct {
	Name      string            `yaml:"name"`
	Enabled   bool              `yaml:"enabled"`
	Transport string            `yaml:"transport"`
	ServerURL string            `yaml:"server_url"`
	Timeout   int               `yaml:"timeout"` // in seconds
	Headers   map[string]string `yaml:"headers"`
}

// MCP holds MCP-specific configurations (now a list of servers)
type MCP struct {
	Servers []MCPServer `yaml:"servers"`
}

// RuleEngine holds rule engine service configurations
type RuleEngine struct {
	EnableRuleEngine    bool   `yaml:"enable_rule_engine"`
	Namespace           string `yaml:"namespace"`
	Kubeconfig          string `yaml:"kubeconfig"`
	MetricsPort         int    `yaml:"metrics_port"`
	AggregationInterval int    `yaml:"aggregation_interval_minutes"` // Interval for aggregating logs in minutes
	RuleCheckInterval   int    `yaml:"rule_check_interval"`          // Interval for checking rules in seconds
}

// AIDecision holds AI decision service configurations
type AIDecision struct {
	EnableAIDecision      bool   `yaml:"enable_ai_decision"`
	Namespace             string `yaml:"namespace"`
	MetricsPort           int    `yaml:"metrics_port"`
	DecisionInterval      int    `yaml:"decision_interval"` // Interval for AI decisions in seconds
	MaxConcurrentRequests int    `yaml:"max_concurrent_requests"`
}

// K8sOperation holds K8s operation service configurations
type K8sOperation struct {
	EnableK8sOperation      bool   `yaml:"enable_k8s_operation"`
	Namespace               string `yaml:"namespace"`
	MetricsPort             int    `yaml:"metrics_port"`
	MaxConcurrentOperations int    `yaml:"max_concurrent_operations"`
}

// Rules holds rule engine configurations
type Rules struct {
	AlertRulesFile       string `yaml:"alert_rules_file"`
	RemediationRulesFile string `yaml:"remediation_rules_file"`
}

// Basic holds basic monitoring configurations
type Basic struct {
	IntervalSeconds int  `yaml:"interval_seconds"`
	MaxRetries      int  `yaml:"max_retries"`
	DryRun          bool `yaml:"dry_run"`
}

// MonitorConfig holds monitor-specific configurations
type MonitorConfig struct {
	Enabled bool   `yaml:"enabled"` // Whether to enable the monitor module
	LogDir  string `yaml:"log_dir"` // Directory where monitor logs are stored
	MCP     struct {
		Port int `yaml:"port"` // Port for the Monitor MCP server
	} `yaml:"mcp"`
}

// MemoryMonitoringConfig holds memory monitoring-specific configurations
type MemoryMonitoringConfig struct {
	Enabled         bool `yaml:"enabled"`          // Whether to enable memory monitoring overall
	IntervalSeconds int  `yaml:"interval_seconds"` // Memory metrics collection interval
	BasicCollection struct {
		Enabled       bool `yaml:"enabled"`
		RetentionDays int  `yaml:"retention_days"`
	} `yaml:"basic_collection"`
	OOMAnalysis struct {
		Enabled        bool `yaml:"enabled"`
		MaxHistoryDays int  `yaml:"max_history_days"`
		MinDataPoints  int  `yaml:"min_data_points"`
	} `yaml:"oom_analysis"`
}

// PVCMonitoringConfig holds PVC monitoring-specific configurations
type PVCMonitoringConfig struct {
	Enabled                   bool `yaml:"enabled"`
	CollectionIntervalSeconds int  `yaml:"collection_interval_seconds"` // 数据采集间隔（秒）
	WarningThresholdPercent   int  `yaml:"warning_threshold_percent"`   // 警告阈值（百分比）
	MaxPodsDisplay            int  `yaml:"max_pods_display"`            // 显示使用的Pod最大数量
	RetentionDays             int  `yaml:"retention_days"`              // 数据保留天数
}

// Monitoring holds monitoring-specific configurations
type Monitoring struct {
	EnableMonitorResources bool                   `yaml:"enable_monitor_resources"`
	EnableMonitorEvents    bool                   `yaml:"enable_monitor_events"`
	Namespaces             []string               `yaml:"namespaces"` // Multiple namespaces for monitoring
	Kubeconfig             string                 `yaml:"kubeconfig"`
	MetricsPort            int                    `yaml:"metrics_port"`
	AggregationInterval    int                    `yaml:"aggregation_interval_minutes"` // Interval for aggregating logs in minutes
	MemoryMonitoring       MemoryMonitoringConfig `yaml:"memory_monitoring"`            // Memory monitoring configuration
}

// Validate validates the configuration parameters
func (c *Config) Validate() error {
	// Validate log level
	if c.LogLevel != "debug" && c.LogLevel != "info" && c.LogLevel != "warn" && c.LogLevel != "error" {
		return fmt.Errorf("invalid log level: %s, must be one of debug, info, warn, error", c.LogLevel)
	}

	// Validate metrics port
	if c.MetricsPort <= 0 || c.MetricsPort > 65535 {
		return fmt.Errorf("invalid metrics port: %d, must be between 1 and 65535", c.MetricsPort)
	}

	// Validate basic configuration
	if c.Basic.IntervalSeconds <= 0 {
		return fmt.Errorf("basic interval seconds must be positive, got: %d", c.Basic.IntervalSeconds)
	}

	if c.Basic.MaxRetries < 0 {
		return fmt.Errorf("basic max retries must be non-negative, got: %d", c.Basic.MaxRetries)
	}

	// Validate monitoring configuration
	if c.Monitoring.EnableMonitorResources || c.Monitoring.EnableMonitorEvents { // Only validate if monitoring or alerting is enabled
		if c.Monitoring.AggregationInterval <= 0 {
			return fmt.Errorf("monitoring aggregation interval must be positive, got: %d", c.Monitoring.AggregationInterval)
		}

		// Validate memory monitoring configuration
		if c.Monitoring.MemoryMonitoring.Enabled {
			if c.Monitoring.MemoryMonitoring.IntervalSeconds <= 0 {
				return fmt.Errorf("memory monitoring interval seconds must be positive, got: %d", c.Monitoring.MemoryMonitoring.IntervalSeconds)
			}

			if c.Monitoring.MemoryMonitoring.BasicCollection.Enabled {
				if c.Monitoring.MemoryMonitoring.BasicCollection.RetentionDays <= 0 {
					return fmt.Errorf("memory monitoring retention days must be positive, got: %d", c.Monitoring.MemoryMonitoring.BasicCollection.RetentionDays)
				}
			}

			if c.Monitoring.MemoryMonitoring.OOMAnalysis.Enabled {
				if c.Monitoring.MemoryMonitoring.OOMAnalysis.MaxHistoryDays <= 0 {
					return fmt.Errorf("OOM analysis max history days must be positive, got: %d", c.Monitoring.MemoryMonitoring.OOMAnalysis.MaxHistoryDays)
				}
				if c.Monitoring.MemoryMonitoring.OOMAnalysis.MinDataPoints <= 0 {
					return fmt.Errorf("OOM analysis min data points must be positive, got: %d", c.Monitoring.MemoryMonitoring.OOMAnalysis.MinDataPoints)
				}
			}
		}

	}

	// Validate rule engine configuration
	if c.RuleEngine.EnableRuleEngine { // Only validate if rule engine is enabled
		if c.RuleEngine.AggregationInterval <= 0 {
			return fmt.Errorf("rule engine aggregation interval must be positive, got: %d", c.RuleEngine.AggregationInterval)
		}

		if c.RuleEngine.RuleCheckInterval <= 0 {
			return fmt.Errorf("rule check interval must be positive, got: %d", c.RuleEngine.RuleCheckInterval)
		}
	}

	// Validate AI decision configuration
	if c.AIDecision.EnableAIDecision { // Only validate if AI decision is enabled
		if c.AIDecision.DecisionInterval <= 0 {
			return fmt.Errorf("AI decision interval must be positive, got: %d", c.AIDecision.DecisionInterval)
		}

		if c.AIDecision.MaxConcurrentRequests <= 0 {
			return fmt.Errorf("AI max concurrent requests must be positive, got: %d", c.AIDecision.MaxConcurrentRequests)
		}
	}

	// Validate K8s operation configuration
	if c.K8sOperation.EnableK8sOperation { // Only validate if K8s operation is enabled
		if c.K8sOperation.MaxConcurrentOperations <= 0 {
			return fmt.Errorf("K8s max concurrent operations must be positive, got: %d", c.K8sOperation.MaxConcurrentOperations)
		}
	}

	return nil
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		// If config file doesn't exist, return default config
		if os.IsNotExist(err) {
			return getDefaultConfig(), nil
		}
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Validate the loaded configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
}

// getDefaultConfig returns a default configuration
func getDefaultConfig() *Config {
	config := &Config{
		LogLevel:    "info",
		MetricsPort: 8080,
		Namespace:   "default",
		Kubeconfig:  "",
		LLM: LLM{
			Model:   "gpt-4o-mini",
			APIKey:  "",
			BaseURL: "",
			Enabled: false,
		},
		MCP: MCP{
			Servers: []MCPServer{
				{
					Name:      "default-mcp-server",
					Enabled:   false,
					Transport: "streamable_http",
					ServerURL: "http://localhost:3000/mcp",
					Timeout:   10,
					Headers:   make(map[string]string),
				},
			},
		},
		Rules: Rules{
			AlertRulesFile:       "./rules/alert_rules.yaml",
			RemediationRulesFile: "./rules/remediation_rules.yaml",
		},
		Basic: Basic{
			IntervalSeconds: 30,
			MaxRetries:      3,
			DryRun:          false,
		},
		Monitoring: Monitoring{
			EnableMonitorResources: true,
			EnableMonitorEvents:    true,
			Namespaces:             []string{"default"}, // Default to single namespace for backward compatibility
			Kubeconfig:             "",
			MetricsPort:            8080,
			AggregationInterval:    10, // Default to 10 minutes
			MemoryMonitoring: MemoryMonitoringConfig{
				Enabled:         true,
				IntervalSeconds: 30, // Default to 30 seconds
				BasicCollection: struct {
					Enabled       bool `yaml:"enabled"`
					RetentionDays int  `yaml:"retention_days"`
				}{
					Enabled:       true,
					RetentionDays: 30,
				},
				OOMAnalysis: struct {
					Enabled        bool `yaml:"enabled"`
					MaxHistoryDays int  `yaml:"max_history_days"`
					MinDataPoints  int  `yaml:"min_data_points"`
				}{
					Enabled:        true,
					MaxHistoryDays: 30,
					MinDataPoints:  10,
				},
			},
		},
		Monitor: MonitorConfig{
			Enabled: true,
			LogDir:  "./logs", // Default log directory relative to executable
			MCP: struct {
				Port int `yaml:"port"`
			}{
				Port: 3001, // Default port for Monitor MCP server
			},
		},
		RuleEngine: RuleEngine{
			EnableRuleEngine:    true,
			Namespace:           "default",
			Kubeconfig:          "",
			MetricsPort:         8082,
			AggregationInterval: 10, // Default to 10 minutes
			RuleCheckInterval:   60, // Default to 60 seconds
		},
		AIDecision: AIDecision{
			EnableAIDecision:      false,
			Namespace:             "default",
			MetricsPort:           8083,
			DecisionInterval:      120, // Default to 120 seconds
			MaxConcurrentRequests: 5,
		},
		K8sOperation: K8sOperation{
			EnableK8sOperation:      true,
			Namespace:               "default",
			MetricsPort:             8084,
			MaxConcurrentOperations: 10,
		},
	}

	// Validate the default config to catch any issues early
	if err := config.Validate(); err != nil {
		// This should not happen with hardcoded defaults, but log if it does
		fmt.Printf("Warning: Default configuration validation failed: %v\n", err)
	}

	return config
}

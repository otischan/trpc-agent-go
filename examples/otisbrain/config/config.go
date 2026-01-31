package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

// Config represents the application configuration
type Config struct {
	LogLevel      string     `yaml:"log_level"`
	MetricsPort   int        `yaml:"metrics_port"`
	Namespace     string     `yaml:"namespace"`
	Kubeconfig    string     `yaml:"kubeconfig"`
	LLM           LLM        `yaml:"llm"`
	MCP           MCP        `yaml:"mcp"`
	Rules         Rules      `yaml:"rules"`
	Basic         Basic      `yaml:"basic"`
	Monitoring    Monitoring `yaml:"monitoring"`
	RuleEngine    RuleEngine `yaml:"rule_engine"`
	AIDecision    AIDecision `yaml:"ai_decision"`
	K8sOperation  K8sOperation `yaml:"k8s_operation"`
}

// LLM holds LLM-specific configurations
type LLM struct {
	Model     string `yaml:"model"`
	APIKey    string `yaml:"api_key"`
	BaseURL   string `yaml:"base_url"`
	Enabled   bool   `yaml:"enabled"`
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
	EnableRuleEngine       bool   `yaml:"enable_rule_engine"`
	Namespace             string `yaml:"namespace"`
	Kubeconfig            string `yaml:"kubeconfig"`
	MetricsPort           int    `yaml:"metrics_port"`
	AggregationInterval   int    `yaml:"aggregation_interval_minutes"` // Interval for aggregating logs in minutes
	RuleCheckInterval     int    `yaml:"rule_check_interval"`         // Interval for checking rules in seconds
}

// AIDecision holds AI decision service configurations
type AIDecision struct {
	EnableAIDecision      bool   `yaml:"enable_ai_decision"`
	Namespace             string `yaml:"namespace"`
	MetricsPort           int    `yaml:"metrics_port"`
	DecisionInterval      int    `yaml:"decision_interval"`            // Interval for AI decisions in seconds
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
	IntervalSeconds int `yaml:"interval_seconds"`
	MaxRetries      int `yaml:"max_retries"`
	DryRun          bool `yaml:"dry_run"`
}

// Monitoring holds monitoring-specific configurations
type Monitoring struct {
	EnableMonitor       bool     `yaml:"enable_monitor"`
	EnableAlert         bool     `yaml:"enable_alert"`
	EnableRemediation   bool     `yaml:"enable_remediation"`
	EnableDecision      bool     `yaml:"enable_decision"`
	Namespace           string   `yaml:"namespace"`           // Single namespace (deprecated, kept for backward compatibility)
	Namespaces          []string `yaml:"namespaces"`          // Multiple namespaces for monitoring
	Kubeconfig          string   `yaml:"kubeconfig"`
	MetricsPort         int      `yaml:"metrics_port"`
	AggregationInterval int      `yaml:"aggregation_interval_minutes"` // Interval for aggregating logs in minutes
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

	return &config, nil
}

// getDefaultConfig returns a default configuration
func getDefaultConfig() *Config {
	return &Config{
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
			EnableMonitor:       true,
			EnableAlert:         true,
			EnableRemediation:   false,
			EnableDecision:      false,
			Namespace:           "default",
			Namespaces:          []string{"default"}, // Default to single namespace for backward compatibility
			Kubeconfig:          "",
			MetricsPort:         8080,
			AggregationInterval: 10, // Default to 10 minutes
		},
		RuleEngine: RuleEngine{
			EnableRuleEngine:      true,
			Namespace:             "default",
			Kubeconfig:            "",
			MetricsPort:           8082,
			AggregationInterval:   10, // Default to 10 minutes
			RuleCheckInterval:     60, // Default to 60 seconds
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
}
package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

// Config represents the application configuration
type Config struct {
	LogLevel      string `yaml:"log_level"`
	MetricsPort   int    `yaml:"metrics_port"`
	Namespace     string `yaml:"namespace"`
	Kubeconfig    string `yaml:"kubeconfig"`
	LLM           LLM    `yaml:"llm"`
	Rules         Rules  `yaml:"rules"`
	Basic         Basic  `yaml:"basic"`
}

// LLM holds LLM-specific configurations
type LLM struct {
	Model     string `yaml:"model"`
	APIKey    string `yaml:"api_key"`
	BaseURL   string `yaml:"base_url"`
	Enabled   bool   `yaml:"enabled"`
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
		LLM: LLM{
			Model:   "gpt-4o-mini",
			Enabled: false,
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
	}
}
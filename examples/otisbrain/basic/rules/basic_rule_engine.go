package rules

import (
	"fmt"
	"log"
)

// Rule defines a basic rule structure
type Rule struct {
	ID          string                 `yaml:"id"`
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Condition   map[string]interface{} `yaml:"condition"`
	Action      string                 `yaml:"action"`
	Severity    string                 `yaml:"severity"`
	Enabled     bool                   `yaml:"enabled"`
}

// BasicRuleEngine handles basic rule processing
type BasicRuleEngine struct {
	rules []Rule
}

// NewBasicRuleEngine creates a new basic rule engine
func NewBasicRuleEngine() *BasicRuleEngine {
	return &BasicRuleEngine{
		rules: []Rule{},
	}
}

// AddRule adds a rule to the engine
func (bre *BasicRuleEngine) AddRule(rule Rule) {
	bre.rules = append(bre.rules, rule)
	log.Printf("Added rule: %s - %s", rule.ID, rule.Name)
}

// Evaluate evaluates rules against the provided data
func (bre *BasicRuleEngine) Evaluate(data map[string]interface{}) []Rule {
	triggeredRules := []Rule{}

	for _, rule := range bre.rules {
		if !rule.Enabled {
			continue
		}

		if bre.evaluateCondition(rule.Condition, data) {
			triggeredRules = append(triggeredRules, rule)
			log.Printf("Rule triggered: %s - %s", rule.ID, rule.Name)
		}
	}

	return triggeredRules
}

// evaluateCondition evaluates a rule condition against the provided data
func (bre *BasicRuleEngine) evaluateCondition(condition map[string]interface{}, data map[string]interface{}) bool {
	// Simple implementation - in a real scenario, this would be more sophisticated
	for key, expectedValue := range condition {
		actualValue, exists := data[key]
		if !exists {
			return false
		}

		// Compare values - this is a simplified comparison
		if fmt.Sprintf("%v", actualValue) != fmt.Sprintf("%v", expectedValue) {
			return false
		}
	}

	return true
}

// GetRules returns all rules
func (bre *BasicRuleEngine) GetRules() []Rule {
	return bre.rules
}
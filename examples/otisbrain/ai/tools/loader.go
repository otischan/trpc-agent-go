package tools

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

// SkillDefinition represents the structure of a skill defined in YAML
type SkillDefinition struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Version     string                 `yaml:"version"`
	Author      string                 `yaml:"author"`
	Category    string                 `yaml:"category"`
	Tags        []string               `yaml:"tags"`
	Inputs      []SkillInput           `yaml:"inputs"`
	Outputs     []SkillOutput          `yaml:"outputs"`
	Execution   SkillExecution         `yaml:"execution"`
	Metadata    map[string]interface{} `yaml:"metadata"`
}

// SkillInput represents an input parameter for a skill
type SkillInput struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Required    bool   `yaml:"required"`
	Default     string `yaml:"default"`
	Description string `yaml:"description"`
}

// SkillOutput represents an output parameter for a skill
type SkillOutput struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
}

// SkillExecution defines how a skill is executed
type SkillExecution struct {
	Type       string                 `yaml:"type"`
	Function   string                 `yaml:"function"`
	Parameters map[string]interface{} `yaml:"parameters"`
}

// SkillLoader handles loading skills from YAML files
type SkillLoader struct {
	skillDir string
}

// NewSkillLoader creates a new skill loader
func NewSkillLoader(skillDir string) *SkillLoader {
	return &SkillLoader{
		skillDir: skillDir,
	}
}

// LoadSkills loads all skills from the skill directory
func (sl *SkillLoader) LoadSkills() (map[string]SkillDefinition, error) {
	skills := make(map[string]SkillDefinition)

	// Check if skill directory exists
	if _, err := os.Stat(sl.skillDir); os.IsNotExist(err) {
		return skills, fmt.Errorf("skill directory does not exist: %s", sl.skillDir)
	}

	// Walk through the skill directory
	err := filepath.Walk(sl.skillDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Process only YAML files
		if !info.IsDir() && (strings.HasSuffix(strings.ToLower(path), ".yaml") || strings.HasSuffix(strings.ToLower(path), ".yml")) {
			skillDef, err := sl.loadSkillFromFile(path)
			if err != nil {
				return fmt.Errorf("failed to load skill from %s: %w", path, err)
			}

			skills[skillDef.Name] = skillDef
		}

		return nil
	})

	return skills, err
}

// loadSkillFromFile loads a skill definition from a YAML file
func (sl *SkillLoader) loadSkillFromFile(filePath string) (SkillDefinition, error) {
	var skillDef SkillDefinition

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return skillDef, fmt.Errorf("failed to read skill file: %w", err)
	}

	err = yaml.Unmarshal(data, &skillDef)
	if err != nil {
		return skillDef, fmt.Errorf("failed to parse skill YAML: %w", err)
	}

	return skillDef, nil
}

// ValidateSkill validates a skill definition
func (sl *SkillLoader) ValidateSkill(skillDef SkillDefinition) error {
	if skillDef.Name == "" {
		return fmt.Errorf("skill name is required")
	}

	if skillDef.Description == "" {
		return fmt.Errorf("skill description is required")
	}

	if skillDef.Execution.Type == "" {
		return fmt.Errorf("skill execution type is required")
	}

	return nil
}
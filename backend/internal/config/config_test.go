package config

import (
	"os"
	"testing"
)

func TestDeriveEnvPrefix(t *testing.T) {
	tests := []struct {
		env      string
		expected string
	}{
		{"development", "dev"},
		{"dev", "dev"},
		{"local", "dev"},
		{"DEVELOPMENT", "dev"},
		{"DEV", "dev"},
		{"staging", "stg"},
		{"stage", "stg"},
		{"stg", "stg"},
		{"STAGING", "stg"},
		{"production", "prd"},
		{"prod", "prd"},
		{"prd", "prd"},
		{"PRODUCTION", "prd"},
		{"PRD", "prd"},
		{"custom", "custom"},
		{"", "dev"},
	}

	for _, tt := range tests {
		actual := DeriveEnvPrefix(tt.env)
		if actual != tt.expected {
			t.Errorf("DeriveEnvPrefix(%q) = %q; expected %q", tt.env, actual, tt.expected)
		}
	}
}

func TestDeriveProjectsCollection(t *testing.T) {
	tests := []struct {
		projectPrefix string
		env           string
		expected      string
	}{
		{"jp", "development", "jp-dev-projects"},
		{"jp", "dev", "jp-dev-projects"},
		{"jp", "staging", "jp-stg-projects"},
		{"jp", "stg", "jp-stg-projects"},
		{"jp", "production", "jp-prd-projects"},
		{"jp", "prd", "jp-prd-projects"},
		{"", "dev", "jp-dev-projects"},
		{"custom", "stg", "custom-stg-projects"},
	}

	for _, tt := range tests {
		actual := DeriveProjectsCollection(tt.projectPrefix, tt.env)
		if actual != tt.expected {
			t.Errorf("DeriveProjectsCollection(%q, %q) = %q; expected %q", tt.projectPrefix, tt.env, actual, tt.expected)
		}
	}
}

func TestLoadProjectsCollectionDefaultsAndOverride(t *testing.T) {
	// 1. Default for development
	_ = os.Setenv("ENV", "development")
	_ = os.Unsetenv("PROJECTS_COLLECTION")
	_ = os.Unsetenv("PROJECT_PREFIX")
	cfg := Load()
	if cfg.ProjectsCollection != "jp-dev-projects" {
		t.Errorf("expected default projects collection 'jp-dev-projects', got %q", cfg.ProjectsCollection)
	}

	// 2. Production
	_ = os.Setenv("ENV", "production")
	cfgProd := Load()
	if cfgProd.ProjectsCollection != "jp-prd-projects" {
		t.Errorf("expected production projects collection 'jp-prd-projects', got %q", cfgProd.ProjectsCollection)
	}

	// 3. Staging
	_ = os.Setenv("ENV", "staging")
	cfgStg := Load()
	if cfgStg.ProjectsCollection != "jp-stg-projects" {
		t.Errorf("expected staging projects collection 'jp-stg-projects', got %q", cfgStg.ProjectsCollection)
	}

	// 4. Custom override via PROJECTS_COLLECTION
	_ = os.Setenv("PROJECTS_COLLECTION", "my-custom-projects")
	cfgOverride := Load()
	if cfgOverride.ProjectsCollection != "my-custom-projects" {
		t.Errorf("expected overridden collection 'my-custom-projects', got %q", cfgOverride.ProjectsCollection)
	}

	// Reset
	_ = os.Unsetenv("PROJECTS_COLLECTION")
	_ = os.Setenv("ENV", "development")
}

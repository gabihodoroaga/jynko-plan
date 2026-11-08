package config

import (
	"os"
	"testing"
)

func TestParseEnv(t *testing.T) {
	tests := []struct {
		env      string
		expected string
	}{
		// Valid aligned values
		{"prd", "prd"},
		{"PRD", "prd"},
		{"stg", "stg"},
		{"STG", "stg"},
		{"dev", "dev"},
		{"DEV", "dev"},

		// Variations like prod or production must NOT be accepted and default to dev
		{"production", "dev"},
		{"prod", "dev"},
		{"staging", "dev"},
		{"stage", "dev"},
		{"development", "dev"},
		{"local", "dev"},
		{"custom", "dev"},
		{"", "dev"},
	}

	for _, tt := range tests {
		actual := ParseEnv(tt.env)
		if actual != tt.expected {
			t.Errorf("ParseEnv(%q) = %q; expected %q", tt.env, actual, tt.expected)
		}
	}
}

func TestDeriveProjectsCollection(t *testing.T) {
	tests := []struct {
		projectPrefix string
		env           string
		expected      string
	}{
		{"jp", "prd", "jp-prd-projects"},
		{"jp", "stg", "jp-stg-projects"},
		{"jp", "dev", "jp-dev-projects"},
		{"", "dev", "jp-dev-projects"},
		{"custom", "stg", "custom-stg-projects"},
		{"custom", "prd", "custom-prd-projects"},
	}

	for _, tt := range tests {
		actual := DeriveProjectsCollection(tt.projectPrefix, tt.env)
		if actual != tt.expected {
			t.Errorf("DeriveProjectsCollection(%q, %q) = %q; expected %q", tt.projectPrefix, tt.env, actual, tt.expected)
		}
	}
}

func TestLoadProjectsCollectionDefaultsAndOverride(t *testing.T) {
	// 1. Default when unset -> dev
	_ = os.Unsetenv("ENV")
	_ = os.Unsetenv("PROJECTS_COLLECTION")
	_ = os.Unsetenv("PROJECT_PREFIX")
	cfgDefault := Load()
	if cfgDefault.Env != "dev" {
		t.Errorf("expected default Env 'dev', got %q", cfgDefault.Env)
	}
	if !cfgDefault.IsDev() || !cfgDefault.IsDevelopment() {
		t.Errorf("expected IsDev() and IsDevelopment() to be true")
	}
	if cfgDefault.ProjectsCollection != "jp-dev-projects" {
		t.Errorf("expected default projects collection 'jp-dev-projects', got %q", cfgDefault.ProjectsCollection)
	}

	// 2. Explicit dev
	_ = os.Setenv("ENV", "dev")
	cfgDev := Load()
	if cfgDev.Env != "dev" || !cfgDev.IsDev() {
		t.Errorf("expected Env 'dev', got %q", cfgDev.Env)
	}
	if cfgDev.ProjectsCollection != "jp-dev-projects" {
		t.Errorf("expected projects collection 'jp-dev-projects', got %q", cfgDev.ProjectsCollection)
	}

	// 3. Explicit prd
	_ = os.Setenv("ENV", "prd")
	cfgPrd := Load()
	if cfgPrd.Env != "prd" || !cfgPrd.IsPrd() || !cfgPrd.IsProduction() {
		t.Errorf("expected Env 'prd', got %q", cfgPrd.Env)
	}
	if cfgPrd.ProjectsCollection != "jp-prd-projects" {
		t.Errorf("expected prd projects collection 'jp-prd-projects', got %q", cfgPrd.ProjectsCollection)
	}

	// 4. Explicit stg
	_ = os.Setenv("ENV", "stg")
	cfgStg := Load()
	if cfgStg.Env != "stg" || !cfgStg.IsStg() {
		t.Errorf("expected Env 'stg', got %q", cfgStg.Env)
	}
	if cfgStg.ProjectsCollection != "jp-stg-projects" {
		t.Errorf("expected stg projects collection 'jp-stg-projects', got %q", cfgStg.ProjectsCollection)
	}

	// 5. Variation like 'production' is rejected and falls back to 'dev'
	_ = os.Setenv("ENV", "production")
	cfgProdVariation := Load()
	if cfgProdVariation.Env != "dev" {
		t.Errorf("expected variation 'production' to fall back to 'dev', got %q", cfgProdVariation.Env)
	}

	// 6. Custom override via PROJECTS_COLLECTION
	_ = os.Setenv("ENV", "prd")
	_ = os.Setenv("PROJECTS_COLLECTION", "my-custom-projects")
	cfgOverride := Load()
	if cfgOverride.ProjectsCollection != "my-custom-projects" {
		t.Errorf("expected overridden collection 'my-custom-projects', got %q", cfgOverride.ProjectsCollection)
	}

	// Reset to dev
	_ = os.Unsetenv("PROJECTS_COLLECTION")
	_ = os.Setenv("ENV", "dev")
}

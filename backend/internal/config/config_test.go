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

	// 5. Variation like 'production' is rejected by ValidateEnv and blocks AllowDevAuth
	_ = os.Setenv("ENV", "production")
	cfgProdVariation := Load()
	if cfgProdVariation.Env != "dev" {
		t.Errorf("expected variation 'production' to fall back to 'dev', got %q", cfgProdVariation.Env)
	}
	if cfgProdVariation.AllowDevAuth() {
		t.Errorf("expected AllowDevAuth() to be false when ENV='production'")
	}
	if err := cfgProdVariation.Validate(); err == nil {
		t.Errorf("expected Validate() to return an error for invalid ENV='production'")
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

func TestCloudRunDevIsolationAndGCSPrefixes(t *testing.T) {
	defer func() {
		_ = os.Unsetenv("K_SERVICE")
		_ = os.Unsetenv("ALLOW_DEV_AUTH")
		_ = os.Unsetenv("ATTACHMENTS_GCS_PREFIX")
		_ = os.Unsetenv("BUILD_SHA")
		_ = os.Unsetenv("BUILD_BRANCH")
		_ = os.Setenv("ENV", "dev")
	}()

	// 1. Local dev (K_SERVICE unset) enables AllowDevAuth and defaults GCS prefix to "dev"
	_ = os.Setenv("ENV", "dev")
	_ = os.Unsetenv("K_SERVICE")
	_ = os.Unsetenv("ALLOW_DEV_AUTH")
	_ = os.Unsetenv("ATTACHMENTS_GCS_PREFIX")
	cfgLocal := Load()
	if !cfgLocal.AllowDevAuth() {
		t.Fatalf("expected AllowDevAuth() == true on local dev when K_SERVICE is unset")
	}
	if cfgLocal.AttachmentsGCSPrefix != "dev" {
		t.Fatalf("expected AttachmentsGCSPrefix 'dev', got %q", cfgLocal.AttachmentsGCSPrefix)
	}

	// 2. Cloud Run dev (K_SERVICE set) disables AllowDevAuth by default while keeping jp-dev-projects
	_ = os.Setenv("K_SERVICE", "jynko-plan-dev")
	_ = os.Setenv("BUILD_SHA", "abc1234")
	_ = os.Setenv("BUILD_BRANCH", "feat/multi-env")
	cfgCloudDev := Load()
	if cfgCloudDev.ProjectsCollection != "jp-dev-projects" {
		t.Fatalf("expected Cloud Run dev to use 'jp-dev-projects', got %q", cfgCloudDev.ProjectsCollection)
	}
	if cfgCloudDev.AllowDevAuth() {
		t.Fatalf("expected AllowDevAuth() == false on Cloud Run (K_SERVICE set) without explicit ALLOW_DEV_AUTH=true")
	}
	if cfgCloudDev.BuildSHA != "abc1234" || cfgCloudDev.BuildBranch != "feat/multi-env" {
		t.Fatalf("expected build metadata to load, got sha=%q branch=%q", cfgCloudDev.BuildSHA, cfgCloudDev.BuildBranch)
	}

	// 3. Staging (ENV=stg) uses "stg" GCS prefix, disables SeedOnStartup, and blocks AllowDevAuth even if ALLOW_DEV_AUTH=true
	_ = os.Setenv("ENV", "stg")
	_ = os.Setenv("ALLOW_DEV_AUTH", "true")
	cfgStg := Load()
	if cfgStg.AllowDevAuth() {
		t.Fatalf("expected AllowDevAuth() == false on ENV=stg even when ALLOW_DEV_AUTH=true")
	}
	if cfgStg.SeedOnStartup {
		t.Fatalf("expected SeedOnStartup == false by default on ENV=stg")
	}
	if cfgStg.AttachmentsGCSPrefix != "stg" {
		t.Fatalf("expected AttachmentsGCSPrefix 'stg' on ENV=stg, got %q", cfgStg.AttachmentsGCSPrefix)
	}

	// 4. Production (ENV=prd) defaults AttachmentsGCSPrefix to empty "" for backward-compatible root objects
	_ = os.Setenv("ENV", "prd")
	cfgPrd := Load()
	if cfgPrd.AttachmentsGCSPrefix != "" {
		t.Fatalf("expected AttachmentsGCSPrefix '' on ENV=prd, got %q", cfgPrd.AttachmentsGCSPrefix)
	}
}

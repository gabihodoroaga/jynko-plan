package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all environment configuration for the Jynko backend server.
type Config struct {
	Port                    string
	Env                     string
	RawEnv                  string
	BuildSHA                string
	BuildBranch             string
	DisableDevAuth          bool
	ProjectPrefix           string
	ProjectsCollection      string
	GoogleCloudProject      string
	FirestoreDatabaseID     string
	IAPAudience             string
	IAPRequireJWT           bool
	SeedOnStartup           bool
	DevUserID               string
	DevUserEmail            string
	DevUserName             string
	CORSAllowedOrigins      []string
	StaticDir               string
	AttachmentsGCSBucket    string
	AttachmentsGCSPrefix    string
	LocalStorageDir         string
	LogLevel                string
	LogFormat               string
	EnableCloudTrace        bool
	EnableCloudMetrics      bool
	EnablePrometheusMetrics bool
	OTelSampleRatio         float64
	ScionHubURL             string
	ScionHubToken           string
	ScionIapAudience        string
	ScionDefaultProjectSlug string
	ScionDefaultTemplate    string
	// PlatformAdminEmails may call cross-project maintenance APIs such as purging old deleted
	// projects (PLAN-13). Comma-separated PLATFORM_ADMIN_EMAILS, normalized to lower case.
	PlatformAdminEmails []string
	// DeletedProjectRetentionDays is the default age a soft-deleted project must reach before the
	// purge API permanently removes it (DELETED_PROJECT_RETENTION_DAYS, default 30).
	DeletedProjectRetentionDays int
}

// Valid aligned environment values: strictly prd, stg, dev.
const (
	EnvPrd = "prd"
	EnvStg = "stg"
	EnvDev = "dev"
)

// ValidateEnv checks that a non-empty environment value is strictly "prd", "stg", or "dev".
func ValidateEnv(raw string) error {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	switch trimmed {
	case "", EnvPrd, EnvStg, EnvDev:
		return nil
	default:
		return fmt.Errorf("invalid ENV %q: must be one of %q, %q, or %q", raw, EnvPrd, EnvStg, EnvDev)
	}
}

// ParseEnv validates and aligns the environment string strictly to "prd", "stg", or "dev".
// Default is "dev". No other variations (such as prod or production) are allowed.
func ParseEnv(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case EnvPrd:
		return EnvPrd
	case EnvStg:
		return EnvStg
	case EnvDev:
		return EnvDev
	default:
		return EnvDev
	}
}

// DeriveProjectsCollection generates the collection name from project prefix and aligned environment:
// e.g. "jp-dev-projects", "jp-stg-projects", "jp-prd-projects".
func DeriveProjectsCollection(projectPrefix, env string) string {
	prefix := strings.ToLower(strings.TrimSpace(projectPrefix))
	if prefix == "" {
		prefix = "jp"
	}
	envVal := ParseEnv(env)
	return fmt.Sprintf("%s-%s-projects", prefix, envVal)
}

// Load reads an optional .env file and populates Config from environment variables.
func Load() *Config {
	loadDotEnv(".env")
	loadDotEnv("backend/.env")

	originsRaw := getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:4202,http://127.0.0.1:4202,http://localhost:4200,http://127.0.0.1:4200")
	var origins []string
	for _, o := range strings.Split(originsRaw, ",") {
		trimmed := strings.TrimSpace(o)
		if trimmed != "" {
			origins = append(origins, trimmed)
		}
	}

	rawEnv, _ := os.LookupEnv("ENV")
	envVal := ParseEnv(getEnv("ENV", EnvDev))
	defaultLogFormat := "json"
	isPrd := envVal == EnvPrd
	isDev := envVal == EnvDev && ValidateEnv(rawEnv) == nil
	kService := strings.TrimSpace(os.Getenv("K_SERVICE"))
	allowDevAuth := getEnvBool("ALLOW_DEV_AUTH", isDev && kService == "")

	projectPrefix := getEnv("PROJECT_PREFIX", "jp")
	projectsCol := getEnv("PROJECTS_COLLECTION", "")
	if projectsCol == "" {
		projectsCol = DeriveProjectsCollection(projectPrefix, envVal)
	}

	defaultGCPProject := ""
	if isDev && kService == "" {
		defaultGCPProject = "codelab1-244421"
	}
	iapAudience := getEnv("IAP_AUDIENCE", "")

	defaultGCSPrefix := ""
	if envVal == EnvDev || envVal == EnvStg {
		defaultGCSPrefix = envVal
	}
	gcsPrefix := defaultGCSPrefix
	if rawPrefix, ok := os.LookupEnv("ATTACHMENTS_GCS_PREFIX"); ok {
		gcsPrefix = strings.Trim(strings.TrimSpace(rawPrefix), "/")
	}

	return &Config{
		Port:                        getEnv("PORT", "8082"),
		Env:                         envVal,
		RawEnv:                      strings.TrimSpace(rawEnv),
		BuildSHA:                    getEnv("BUILD_SHA", ""),
		BuildBranch:                 getEnv("BUILD_BRANCH", ""),
		DisableDevAuth:              !allowDevAuth,
		ProjectPrefix:               projectPrefix,
		ProjectsCollection:          projectsCol,
		GoogleCloudProject:          getEnv("GOOGLE_CLOUD_PROJECT", defaultGCPProject),
		FirestoreDatabaseID:         getEnv("FIRESTORE_DATABASE_ID", "(default)"),
		IAPAudience:                 iapAudience,
		IAPRequireJWT:               getEnvBool("IAP_REQUIRE_JWT", iapAudience != ""),
		SeedOnStartup:               getEnvBool("SEED_ON_STARTUP", isDev),
		DevUserID:                   getEnv("DEV_USER_ID", "dev-user-1"),
		DevUserEmail:                getEnv("DEV_USER_EMAIL", "dev@jynko.local"),
		DevUserName:                 getEnv("DEV_USER_NAME", "Local Developer"),
		CORSAllowedOrigins:          origins,
		StaticDir:                   getEnv("STATIC_DIR", ""),
		AttachmentsGCSBucket:        getEnv("ATTACHMENTS_GCS_BUCKET", ""),
		AttachmentsGCSPrefix:        gcsPrefix,
		LocalStorageDir:             getEnv("LOCAL_STORAGE_DIR", "data/attachments"),
		LogLevel:                    getEnv("LOG_LEVEL", "info"),
		LogFormat:                   getEnv("LOG_FORMAT", defaultLogFormat),
		EnableCloudTrace:            getEnvBool("ENABLE_CLOUD_TRACE", isPrd),
		EnableCloudMetrics:          getEnvBool("ENABLE_CLOUD_METRICS", isPrd),
		EnablePrometheusMetrics:     getEnvBool("ENABLE_PROMETHEUS_METRICS", true),
		OTelSampleRatio:             getEnvFloat("OTEL_SAMPLE_RATIO", 1.0),
		ScionHubURL:                 strings.TrimRight(getEnv("SCION_HUB_URL", ""), "/"),
		ScionHubToken:               getEnv("SCION_HUB_TOKEN", ""),
		ScionIapAudience:            getEnv("SCION_IAP_AUDIENCE", ""),
		ScionDefaultProjectSlug:     getEnv("SCION_DEFAULT_PROJECT_SLUG", "global"),
		ScionDefaultTemplate:        getEnv("SCION_DEFAULT_TEMPLATE", "jynko-dev"),
		PlatformAdminEmails:         parseEmailList(getEnv("PLATFORM_ADMIN_EMAILS", "")),
		DeletedProjectRetentionDays: getEnvInt("DELETED_PROJECT_RETENTION_DAYS", 30),
	}
}

// IsPlatformAdmin reports whether email is listed in PLATFORM_ADMIN_EMAILS (case-insensitive).
func (c *Config) IsPlatformAdmin(email string) bool {
	if c == nil {
		return false
	}
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return false
	}
	for _, e := range c.PlatformAdminEmails {
		if e == normalized {
			return true
		}
	}
	return false
}

func parseEmailList(raw string) []string {
	out := []string{}
	for _, part := range strings.Split(raw, ",") {
		if e := strings.ToLower(strings.TrimSpace(part)); e != "" {
			out = append(out, e)
		}
	}
	return out
}

// IsScionConfigured reports whether Scion Hub URL and token are both configured.
func (c *Config) IsScionConfigured() bool {
	return c != nil && strings.TrimSpace(c.ScionHubURL) != "" && strings.TrimSpace(c.ScionHubToken) != ""
}

// Validate checks startup invariants and fails closed on misconfiguration.
func (c *Config) Validate() error {
	if err := ValidateEnv(c.RawEnv); err != nil {
		return err
	}
	if !c.AllowDevAuth() && strings.TrimSpace(c.GoogleCloudProject) == "" {
		return fmt.Errorf("GOOGLE_CLOUD_PROJECT must be set when ENV=%q", c.Env)
	}
	return nil
}

// AllowDevAuth returns true strictly when local/explicit dev auth is enabled on ENV="dev"
// (disabled automatically on Cloud Run when K_SERVICE is set unless ALLOW_DEV_AUTH=true, and never on invalid RawEnv).
func (c *Config) AllowDevAuth() bool {
	if c == nil || c.Env != EnvDev || c.DisableDevAuth {
		return false
	}
	return ValidateEnv(c.RawEnv) == nil
}

// IsPrd returns true when running in prd mode.
func (c *Config) IsPrd() bool {
	return c.Env == EnvPrd
}

// IsProduction is an alias for IsPrd (checking Env == "prd").
func (c *Config) IsProduction() bool {
	return c.IsPrd()
}

// IsStg returns true when running in stg mode.
func (c *Config) IsStg() bool {
	return c.Env == EnvStg
}

// IsDev returns true when running in dev mode.
func (c *Config) IsDev() bool {
	return c.Env == EnvDev
}

// IsDevelopment is an alias for IsDev (checking Env == "dev").
func (c *Config) IsDevelopment() bool {
	return c.IsDev()
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if val, ok := os.LookupEnv(key); ok && strings.TrimSpace(val) != "" {
		b, err := strconv.ParseBool(strings.TrimSpace(val))
		if err == nil {
			return b
		}
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val, ok := os.LookupEnv(key); ok && strings.TrimSpace(val) != "" {
		n, err := strconv.Atoi(strings.TrimSpace(val))
		if err == nil && n >= 0 {
			return n
		}
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if val, ok := os.LookupEnv(key); ok && strings.TrimSpace(val) != "" {
		f, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
		if err == nil {
			return f
		}
	}
	return fallback
}

func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, val)
		}
	}
}

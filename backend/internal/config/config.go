package config

import (
	"bufio"
	"os"
	"strings"
)

// Config holds all environment configuration for the Jynko backend server.
type Config struct {
	Port                string
	Env                 string
	GoogleCloudProject  string
	FirestoreDatabaseID string
	IAPAudience         string
	DevUserID           string
	DevUserEmail        string
	DevUserName         string
	CORSAllowedOrigins  []string
	StaticDir           string
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

	return &Config{
		Port:                getEnv("PORT", "8082"),
		Env:                 getEnv("ENV", "development"),
		GoogleCloudProject:  getEnv("GOOGLE_CLOUD_PROJECT", "codelab1-244421"),
		FirestoreDatabaseID: getEnv("FIRESTORE_DATABASE_ID", "(default)"),
		IAPAudience:         getEnv("IAP_AUDIENCE", ""),
		DevUserID:           getEnv("DEV_USER_ID", "dev-user-1"),
		DevUserEmail:        getEnv("DEV_USER_EMAIL", "dev@jynko.local"),
		DevUserName:         getEnv("DEV_USER_NAME", "Local Developer"),
		CORSAllowedOrigins:  origins,
		StaticDir:           getEnv("STATIC_DIR", ""),
	}
}

// IsDevelopment returns true when running in local development mode.
func (c *Config) IsDevelopment() bool {
	return strings.EqualFold(c.Env, "development") || strings.EqualFold(c.Env, "dev") || strings.EqualFold(c.Env, "local")
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
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

package config

import "os"

type Config struct {
	ServerAddr string
	JWTSecret  string
	DBDriver   string
	DBDSN      string
	PluginsDir string
}

func Load() Config {
	cfg := Config{
		ServerAddr: getEnv("SERVER_ADDR", ":8080"),
		JWTSecret:  getEnv("JWT_SECRET", "dev-secret-change-me"),
		DBDriver:   getEnv("DB_DRIVER", "sqlite"),
		DBDSN:      getEnv("DB_DSN", "skoll2.db"),
		PluginsDir: getEnv("PLUGINS_DIR", "../plugins"),
	}
	return cfg
}

func getEnv(key, defaultValue string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	return val
}

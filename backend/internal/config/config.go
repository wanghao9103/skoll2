package config

import (
	"os"
	"strconv"
)

type Config struct {
	ServerAddr string
	JWTSecret  string
	DBDriver   string
	DBDSN      string
	PluginsDir string

	PluginDefaultChannel            string
	PluginStartupStrategy           string
	PluginProcessStartupTimeoutMs   int
	PluginProcessRequestTimeoutMs   int
	PluginProcessIdleRecycleSeconds int
	PluginProcessMaxIdleConnsHost   int
}

func Load() Config {
	cfg := Config{
		ServerAddr: getEnv("SERVER_ADDR", ":8080"),
		JWTSecret:  getEnv("JWT_SECRET", "dev-secret-change-me"),
		DBDriver:   getEnv("DB_DRIVER", "sqlite"),
		DBDSN:      getEnv("DB_DSN", "skoll2.db"),
		PluginsDir: getEnv("PLUGINS_DIR", "../plugins"),

		PluginDefaultChannel:            getEnv("PLUGIN_DEFAULT_CHANNEL", "js"),
		PluginStartupStrategy:           getEnv("PLUGIN_STARTUP_STRATEGY", "lazy"),
		PluginProcessStartupTimeoutMs:   getEnvInt("PLUGIN_PROCESS_STARTUP_TIMEOUT_MS", 2000),
		PluginProcessRequestTimeoutMs:   getEnvInt("PLUGIN_PROCESS_REQUEST_TIMEOUT_MS", 3000),
		PluginProcessIdleRecycleSeconds: getEnvInt("PLUGIN_PROCESS_IDLE_RECYCLE_SECONDS", 180),
		PluginProcessMaxIdleConnsHost:   getEnvInt("PLUGIN_PROCESS_MAX_IDLE_CONNS_PER_HOST", 2),
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

func getEnvInt(key string, defaultValue int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return n
}

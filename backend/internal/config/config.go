package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ConfigFile string

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

type fileConfig struct {
	Server struct {
		Addr string `yaml:"addr"`
	} `yaml:"server"`
	Auth struct {
		JWTSecret string `yaml:"jwtSecret"`
	} `yaml:"auth"`
	Database struct {
		Driver string `yaml:"driver"`
		DSN    string `yaml:"dsn"`
	} `yaml:"database"`
	Plugins struct {
		Dir     string `yaml:"dir"`
		Process struct {
			DefaultChannel      string `yaml:"defaultChannel"`
			StartupStrategy     string `yaml:"startupStrategy"`
			StartupTimeoutMs    int    `yaml:"startupTimeoutMs"`
			RequestTimeoutMs    int    `yaml:"requestTimeoutMs"`
			IdleRecycleSeconds  int    `yaml:"idleRecycleSeconds"`
			MaxIdleConnsPerHost int    `yaml:"maxIdleConnsPerHost"`
		} `yaml:"process"`
	} `yaml:"plugins"`
}

func Load() Config {
	cfg := defaultConfig()

	configFile := stringsTrimSpace(getEnv("CONFIG_FILE", ""))
	if configFile == "" {
		configFile = filepath.Clean("./configs/config.yaml")
	}
	cfg.ConfigFile = configFile
	_ = loadFromFile(&cfg, configFile)

	// Environment variables override config file values.
	cfg.ServerAddr = getEnv("SERVER_ADDR", cfg.ServerAddr)
	cfg.JWTSecret = getEnv("JWT_SECRET", cfg.JWTSecret)
	cfg.DBDriver = getEnv("DB_DRIVER", cfg.DBDriver)
	cfg.DBDSN = getEnv("DB_DSN", cfg.DBDSN)
	cfg.PluginsDir = getEnv("PLUGINS_DIR", cfg.PluginsDir)

	cfg.PluginDefaultChannel = getEnv("PLUGIN_DEFAULT_CHANNEL", cfg.PluginDefaultChannel)
	cfg.PluginStartupStrategy = getEnv("PLUGIN_STARTUP_STRATEGY", cfg.PluginStartupStrategy)
	cfg.PluginProcessStartupTimeoutMs = getEnvInt("PLUGIN_PROCESS_STARTUP_TIMEOUT_MS", cfg.PluginProcessStartupTimeoutMs)
	cfg.PluginProcessRequestTimeoutMs = getEnvInt("PLUGIN_PROCESS_REQUEST_TIMEOUT_MS", cfg.PluginProcessRequestTimeoutMs)
	cfg.PluginProcessIdleRecycleSeconds = getEnvInt("PLUGIN_PROCESS_IDLE_RECYCLE_SECONDS", cfg.PluginProcessIdleRecycleSeconds)
	cfg.PluginProcessMaxIdleConnsHost = getEnvInt("PLUGIN_PROCESS_MAX_IDLE_CONNS_PER_HOST", cfg.PluginProcessMaxIdleConnsHost)

	return cfg
}

func defaultConfig() Config {
	return Config{
		ConfigFile: "",
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
}

func loadFromFile(cfg *Config, filePath string) error {
	if cfg == nil || stringsTrimSpace(filePath) == "" {
		return nil
	}
	raw, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	f := fileConfig{}
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return err
	}

	if v := stringsTrimSpace(f.Server.Addr); v != "" {
		cfg.ServerAddr = v
	}
	if v := stringsTrimSpace(f.Auth.JWTSecret); v != "" {
		cfg.JWTSecret = v
	}
	if v := stringsTrimSpace(f.Database.Driver); v != "" {
		cfg.DBDriver = v
	}
	if v := stringsTrimSpace(f.Database.DSN); v != "" {
		cfg.DBDSN = v
	}
	if v := stringsTrimSpace(f.Plugins.Dir); v != "" {
		cfg.PluginsDir = v
	}

	if v := stringsTrimSpace(f.Plugins.Process.DefaultChannel); v != "" {
		cfg.PluginDefaultChannel = v
	}
	if v := stringsTrimSpace(f.Plugins.Process.StartupStrategy); v != "" {
		cfg.PluginStartupStrategy = v
	}
	if f.Plugins.Process.StartupTimeoutMs > 0 {
		cfg.PluginProcessStartupTimeoutMs = f.Plugins.Process.StartupTimeoutMs
	}
	if f.Plugins.Process.RequestTimeoutMs > 0 {
		cfg.PluginProcessRequestTimeoutMs = f.Plugins.Process.RequestTimeoutMs
	}
	if f.Plugins.Process.IdleRecycleSeconds > 0 {
		cfg.PluginProcessIdleRecycleSeconds = f.Plugins.Process.IdleRecycleSeconds
	}
	if f.Plugins.Process.MaxIdleConnsPerHost > 0 {
		cfg.PluginProcessMaxIdleConnsHost = f.Plugins.Process.MaxIdleConnsPerHost
	}

	return nil
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

func stringsTrimSpace(v string) string {
	return strings.TrimSpace(v)
}

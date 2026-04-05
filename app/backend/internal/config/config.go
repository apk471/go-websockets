package config

import (
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	_ "github.com/joho/godotenv/autoload"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
	"github.com/rs/zerolog"
)

type Config struct {
	Primary       Primary              `koanf:"primary" validate:"required"`
	Server        ServerConfig         `koanf:"server" validate:"required"`
	Database      DatabaseConfig       `koanf:"database" validate:"required"`
	Auth          AuthConfig           `koanf:"auth" validate:"required"`
	Redis         RedisConfig          `koanf:"redis" validate:"required"`
	Integration   IntegrationConfig    `koanf:"integration" validate:"required"`
	Observability *ObservabilityConfig `koanf:"observability"`
}

type Primary struct {
	Env string `koanf:"env" validate:"required"`
}

type ServerConfig struct {
	Port               string   `koanf:"port" validate:"required"`
	ReadTimeout        int      `koanf:"read_timeout" validate:"required"`
	WriteTimeout       int      `koanf:"write_timeout" validate:"required"`
	IdleTimeout        int      `koanf:"idle_timeout" validate:"required"`
	CORSAllowedOrigins []string `koanf:"cors_allowed_origins" validate:"required"`
	BodyLimit          string   `koanf:"body_limit"`
	HTTPRateLimitRPS   float64  `koanf:"http_rate_limit_rps"`
	HTTPRateLimitBurst int      `koanf:"http_rate_limit_burst"`
	HTTPRateLimitTTL   int      `koanf:"http_rate_limit_ttl_seconds"`
	WSHandshakeTimeout int      `koanf:"ws_handshake_timeout_seconds"`
	WSMaxConnectionsIP int      `koanf:"ws_max_connections_per_ip"`
	WSMaxMessageBytes  int64    `koanf:"ws_max_message_bytes"`
	WSPingInterval     int      `koanf:"ws_ping_interval_seconds"`
	WSPongWait         int      `koanf:"ws_pong_wait_seconds"`
	WSWriteTimeout     int      `koanf:"ws_write_timeout_seconds"`
	WSUpgradeRateRPS   float64  `koanf:"ws_upgrade_rate_limit_rps"`
	WSUpgradeRateBurst int      `koanf:"ws_upgrade_rate_limit_burst"`
	WSMessageRateRPS   float64  `koanf:"ws_message_rate_limit_rps"`
	WSMessageRateBurst int      `koanf:"ws_message_rate_limit_burst"`
}

// Database configuration
type DatabaseConfig struct {
	Host            string `koanf:"host" validate:"required"`
	Port            int    `koanf:"port" validate:"required"`
	User            string `koanf:"user" validate:"required"`
	Password        string `koanf:"password"`
	Name            string `koanf:"name" validate:"required"`
	SSLMode         string `koanf:"ssl_mode" validate:"required"`
	MaxOpenConns    int    `koanf:"max_open_conns" validate:"required"`
	MaxIdleConns    int    `koanf:"max_idle_conns" validate:"required"`
	ConnMaxLifetime int    `koanf:"conn_max_lifetime" validate:"required"`
	ConnMaxIdleTime int    `koanf:"conn_max_idle_time" validate:"required"`
}
type RedisConfig struct {
	Address string `koanf:"address" validate:"required"`
}

type IntegrationConfig struct {
	ResendAPIKey string `koanf:"resend_api_key" validate:"required"`
}

type AuthConfig struct {
	SecretKey string `koanf:"secret_key" validate:"required"`
}

func LoadConfig() (*Config, error) {
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()

	k := koanf.New(".")

	err := k.Load(env.Provider("BOILERPLATE_", ".", func(s string) string {
		return strings.ToLower(strings.TrimPrefix(s, "BOILERPLATE_"))
	}), nil)
	if err != nil {
		logger.Fatal().Err(err).Msg("could not load initial env variables")
	}

	mainConfig := &Config{}

	err = k.Unmarshal("", mainConfig)
	if err != nil {
		logger.Fatal().Err(err).Msg("could not unmarshal main config")
	}

	validate := validator.New()

	err = validate.Struct(mainConfig)
	if err != nil {
		logger.Fatal().Err(err).Msg("config validation failed")
	}

	// Set default observability config if not provided
	if mainConfig.Observability == nil {
		mainConfig.Observability = DefaultObservabilityConfig()
	}

	applyServerSecurityDefaults(&mainConfig.Server)

	// Override service name and environment from primary config
	mainConfig.Observability.ServiceName = "boilerplate"
	mainConfig.Observability.Environment = mainConfig.Primary.Env

	// Validate observability config
	if err := mainConfig.Observability.Validate(); err != nil {
		logger.Fatal().Err(err).Msg("invalid observability config")
	}

	return mainConfig, nil
}

func applyServerSecurityDefaults(cfg *ServerConfig) {
	if cfg.BodyLimit == "" {
		cfg.BodyLimit = "1M"
	}

	if cfg.HTTPRateLimitRPS <= 0 {
		cfg.HTTPRateLimitRPS = 10
	}
	if cfg.HTTPRateLimitBurst <= 0 {
		cfg.HTTPRateLimitBurst = 20
	}
	if cfg.HTTPRateLimitTTL <= 0 {
		cfg.HTTPRateLimitTTL = 300
	}

	if cfg.WSHandshakeTimeout <= 0 {
		cfg.WSHandshakeTimeout = 5
	}
	if cfg.WSMaxConnectionsIP <= 0 {
		cfg.WSMaxConnectionsIP = 5
	}
	if cfg.WSMaxMessageBytes <= 0 {
		cfg.WSMaxMessageBytes = 4096
	}
	if cfg.WSPingInterval <= 0 {
		cfg.WSPingInterval = 25
	}
	if cfg.WSPongWait <= 0 {
		cfg.WSPongWait = 60
	}
	if cfg.WSWriteTimeout <= 0 {
		cfg.WSWriteTimeout = 10
	}
	if cfg.WSUpgradeRateRPS <= 0 {
		cfg.WSUpgradeRateRPS = 0.5
	}
	if cfg.WSUpgradeRateBurst <= 0 {
		cfg.WSUpgradeRateBurst = 3
	}
	if cfg.WSMessageRateRPS <= 0 {
		cfg.WSMessageRateRPS = 1
	}
	if cfg.WSMessageRateBurst <= 0 {
		cfg.WSMessageRateBurst = 5
	}
}

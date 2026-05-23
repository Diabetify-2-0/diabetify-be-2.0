package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Settings struct {
	AppEnv      string
	Port        string
	UseSharding bool
	DB          DBConfig
	JWT         JWTConfig
	RabbitMQ    RabbitMQConfig
	Redis       RedisConfig
	SMTP        SMTPConfig
	OpenAI      OpenAIConfig
	CORS        CORSConfig
	MLOps       MLOpsConfig
	Google      GoogleConfig
}

type DBConfig struct {
	Host     string
	Host2    string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type JWTConfig struct {
	Secret string
}

type RabbitMQConfig struct {
	URL string
}

type RedisConfig struct {
	URL      string
	Host     string
	Port     string
	Password string
	DB       int
}

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	Sender   string
}

type OpenAIConfig struct {
	APIKey string
}

type CORSConfig struct {
	AllowedOrigins []string
}

type MLOpsConfig struct {
	ServiceURL string
}

type GoogleConfig struct {
	ClientIDs   []string
	Secret      string
	CallbackURL string
}

func Load() Settings {
	return Settings{
		AppEnv:      env("APP_ENV", "dev"),
		Port:        env("PORT", "8080"),
		UseSharding: boolEnv("USE_SHARDING", false),
		DB: DBConfig{
			Host:     env("DB_HOST", "localhost"),
			Host2:    strings.TrimSpace(os.Getenv("DB_HOST2")),
			Port:     env("DB_PORT", "5432"),
			User:     env("DB_USER", "postgres"),
			Password: env("DB_PASSWORD", "postgres"),
			Name:     env("DB_NAME", "diabetify"),
			SSLMode:  env("DB_SSLMODE", "disable"),
		},
		JWT: JWTConfig{
			Secret: strings.TrimSpace(os.Getenv("JWT_SECRET_KEY")),
		},
		RabbitMQ: RabbitMQConfig{
			URL: env("RABBITMQ_URL", "amqp://admin:password123@localhost:5672/"),
		},
		Redis: RedisConfig{
			URL:      strings.TrimSpace(os.Getenv("REDIS_URL")),
			Host:     env("REDIS_HOST", "localhost"),
			Port:     env("REDIS_PORT", "6379"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       intEnv("REDIS_DB", 0),
		},
		SMTP: SMTPConfig{
			Host:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
			Port:     strings.TrimSpace(os.Getenv("SMTP_PORT")),
			Username: strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
			Password: os.Getenv("SMTP_PASSWORD"),
			Sender:   strings.TrimSpace(os.Getenv("SMTP_SENDER")),
		},
		OpenAI: OpenAIConfig{
			APIKey: strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
		},
		CORS: CORSConfig{
			AllowedOrigins: splitCSV(env("ALLOWED_ORIGINS", "http://localhost:3000")),
		},
		MLOps: MLOpsConfig{
			ServiceURL: strings.TrimRight(env("MLOPS_SERVICE_URL", "http://localhost:8000"), "/"),
		},
		Google: GoogleConfig{
			ClientIDs:   splitCSV(os.Getenv("GOOGLE_KEY")),
			Secret:      os.Getenv("GOOGLE_SECRET"),
			CallbackURL: os.Getenv("GOOGLE_CALLBACK_URL"),
		},
	}
}

func (s Settings) ValidateRuntime() error {
	if strings.TrimSpace(s.JWT.Secret) == "" {
		return fmt.Errorf("JWT_SECRET_KEY environment variable is not set")
	}

	if s.IsProduction() {
		if s.JWT.Secret == "diabetify-dev-secret-change-me" {
			return fmt.Errorf("production APP_ENV requires a non-default JWT_SECRET_KEY")
		}
		parsedRabbitMQ, err := url.Parse(s.RabbitMQ.URL)
		if err == nil && parsedRabbitMQ.User != nil {
			password, _ := parsedRabbitMQ.User.Password()
			if parsedRabbitMQ.User.Username() == "admin" && password == "password123" {
				return fmt.Errorf("production APP_ENV requires non-default RabbitMQ credentials")
			}
		}
	}

	return nil
}

func (s Settings) IsProduction() bool {
	envName := strings.ToLower(strings.TrimSpace(s.AppEnv))
	return envName == "prod" || envName == "production"
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func boolEnv(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

func intEnv(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

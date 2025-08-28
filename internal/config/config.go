package config

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	AppConfig              AppConfig              `mapstructure:"AppConfig"`
	DatabaseConfig         DatabaseConfig         `mapstructure:"DatabaseConfig"`
	RedisConfig            RedisConfig            `mapstructure:"RedisConfig"`
	AuthConfig             AuthConfig             `mapstructure:"AuthConfig"`
	EmailConfig            EmailConfig            `mapstructure:"EmailConfig"`
	AWSConfig              AWSConfig              `mapstructure:"AWSConfig"`
	InterviewSessionConfig InterviewSessionConfig `mapstructure:"InterviewSessionConfig"`
}

type AppConfig struct {
	AppPort     string   `mapstructure:"APP_PORT"`
	AppEnv      string   `mapstructure:"APP_ENV"`
	APIHost     string   `mapstructure:"API_HOST"`
	CORSOrigins []string `mapstructure:"CORS_ORIGINS"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"POSTGRES_HOST"`
	Port     string `mapstructure:"POSTGRES_PORT"`
	User     string `mapstructure:"POSTGRES_USER"`
	Password string `mapstructure:"POSTGRES_PASSWORD"`
	DbName   string `mapstructure:"POSTGRES_DB"`
}

type RedisConfig struct {
	Host     string `mapstructure:"REDIS_HOST"`
	Port     string `mapstructure:"REDIS_PORT"`
	Password string `mapstructure:"REDIS_PASSWORD"`
	DBTemp   int    `mapstructure:"REDIS_DB_TEMP"`
	DBQueue  int    `mapstructure:"REDIS_DB_QUEUE"`
}

type AuthConfig struct {
	EncryptionSecretKey        string        `mapstructure:"ENCRYPTION_SECRET_KEY"`
	AccessTokenDuration        time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	RefreshTokenDuration       time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`
	ResetPasswordTokenDuration time.Duration `mapstructure:"RESET_PASSWORD_TOKEN_DURATION"`
	TokenGraceWindow           time.Duration `mapstructure:"TOKEN_GRACE_WINDOW"`
	CookieDomain               string        `mapstructure:"COOKIE_DOMAIN"`
	CookieRejectHTTP           bool          `mapstructure:"COOKIE_REJECT_HTTP"`
}

type EmailConfig struct {
	Host                     string        `mapstructure:"EMAIL_HOST"`
	Port                     int           `mapstructure:"EMAIL_PORT"`
	Username                 string        `mapstructure:"EMAIL_USERNAME"`
	Password                 string        `mapstructure:"EMAIL_PASSWORD"`
	From                     string        `mapstructure:"EMAIL_FROM"`
	ResetPasswordURL         string        `mapstructure:"RESET_PASSWORD_URL"`
	VerifyEmailTokenDuration time.Duration `mapstructure:"VERIFY_EMAIL_TOKEN_DURATION"`
	EncryptionSecretKey      string        `mapstructure:"ENCRYPTION_SECRET_KEY"`
}

type AWSConfig struct {
	Region             string        `mapstructure:"AWS_REGION"`
	S3Bucket           string        `mapstructure:"AWS_S3_BUCKET"`
	S3AccessKey        string        `mapstructure:"AWS_S3_ACCESS_KEY"`
	S3SecretAccessKey  string        `mapstructure:"AWS_S3_SECRET_ACCESS_KEY"`
	PresignedURLExpiry time.Duration `mapstructure:"AWS_S3_PRESIGNED_URL_EXPIRY"`
}

type InterviewSessionConfig struct {
	InterviewAgentURL        string        `mapstructure:"INTERVIEW_AGENT_URL"`
	InterviewWebsocketPath   string        `mapstructure:"INTERVIEW_WEBSOCKET_PATH"`
	InterviewSessionTokenTTL time.Duration `mapstructure:"INTERVIEW_SESSION_TOKEN_TTL"`
	InterviewSessionDuration time.Duration `mapstructure:"INTERVIEW_SESSION_DURATION"`
	EncryptionSecretKey      string        `mapstructure:"ENCRYPTION_SECRET_KEY"`
}

func LoadConfig() (*Config, error) {
	v := viper.New()

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if path := os.Getenv("CONFIG_PATH"); path != "" {
		v.SetConfigFile(path)
	} else {
		env := os.Getenv("ENV")
		if env == "" {
			env = "dev"
		}
		v.SetConfigName("config." + env)
		v.SetConfigType("yaml")
		v.AddConfigPath("./config")
	}

	if err := v.ReadInConfig(); err != nil {
		log.Printf("[config] No config file found, using ENV/defaults only: %v", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

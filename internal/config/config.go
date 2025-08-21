package config

import (
	"log"
	"os"
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
	AppPort string `mapstructure:"APP_PORT"`
	AppEnv  string `mapstructure:"APP_ENV"`
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
	JwtSecretKey               string        `mapstructure:"JWT_SECRET_KEY"`
	AccessTokenDuration        time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	RefreshTokenDuration       time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`
	ResetPasswordTokenDuration time.Duration `mapstructure:"RESET_PASSWORD_TOKEN_DURATION"`
	VerifyEmailTokenDuration   time.Duration `mapstructure:"VERIFY_EMAIL_TOKEN_DURATION"`
	TokenGraceWindow           time.Duration `mapstructure:"TOKEN_GRACE_WINDOW"`
	CookieDomain               string        `mapstructure:"COOKIE_DOMAIN"`
	CookieRejectHTTP           bool          `mapstructure:"COOKIE_REJECT_HTTP"`
}

type EmailConfig struct {
	Host             string `mapstructure:"EMAIL_HOST"`
	Port             int    `mapstructure:"EMAIL_PORT"`
	Username         string `mapstructure:"EMAIL_USERNAME"`
	Password         string `mapstructure:"EMAIL_PASSWORD"`
	From             string `mapstructure:"EMAIL_FROM"`
	ResetPasswordURL string `mapstructure:"RESET_PASSWORD_URL"`
}

type AWSConfig struct {
	Region             string        `mapstructure:"AWS_REGION"`
	S3Bucket           string        `mapstructure:"AWS_S3_BUCKET"`
	S3AccessKey        string        `mapstructure:"AWS_S3_ACCESS_KEY"`
	S3SecretAccessKey  string        `mapstructure:"AWS_S3_SECRET_ACCESS_KEY"`
	PresignedURLExpiry time.Duration `mapstructure:"AWS_S3_PRESIGNED_URL_EXPIRY"`
}

type InterviewSessionConfig struct {
	InterviewAgentURL             string        `mapstructure:"INTERVIEW_AGENT_URL"`
	InterviewSessionTokenDuration time.Duration `mapstructure:"INTERVIEW_SESSION_TOKEN_DURATION"`
}

func LoadConfig() (*Config, error) {

	env := os.Getenv("ENV")
	if env == "" {
		env = "dev"
	}

	viper.SetConfigName("config." + env)

	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %v", err)
		return nil, err
	}

	var config = &Config{}
	if err := viper.Unmarshal(config); err != nil {
		log.Fatalf("Unable to unmarshal config: %v", err)
		return nil, err
	}

	return config, nil
}

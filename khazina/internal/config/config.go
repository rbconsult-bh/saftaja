package config

import (
	"encoding/base64"
	"errors"
	"reflect"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Config struct {
	DBHost             string `mapstructure:"DB_HOST"`
	DBPort             uint16 `mapstructure:"DB_PORT"`
	DBDatabase         string `mapstructure:"DB_DATABASE"`
	DBUser             string `mapstructure:"DB_USER"`
	DBPassword         string `mapstructure:"DB_PASSWORD"`
	Port               int    `mapstructure:"PORT"`
	EncryptionKey      string `mapstructure:"ENCRYPTION_KEY"`
	VerifyDomainSecret string `mapstructure:"VERIFY_DOMAIN_SECRET"`
	AdminAPIKey        string `mapstructure:"ADMIN_API_KEY"`
	Emailer            string `mapstructure:"EMAILER"`
	ResendAPIKey       string `mapstructure:"RESEND_API_KEY"`
	Domain             string `mapstructure:"DOMAIN"`
	JWTPrivateKey      string `mapstructure:"JWT_PRIVATE_KEY"`
}

func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(configPath)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}

		log.Warn().Err(err).Str("config_path", configPath).Msg("no .env file found, using environments variables only")
	}

	t := reflect.TypeOf(Config{})
	for i := 0; i < t.NumField(); i++ {
		if tag := t.Field(i).Tag.Get("mapstructure"); tag != "" {
			if err := viper.BindEnv(tag); err != nil {
				log.Warn().Err(err).Str("config_tag", tag).Msg("failed to bind env")
				continue
			}
		}
	}

	viper.AutomaticEnv()

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.DBHost == "" {
		return errors.New("DB_HOST is required")
	}
	if c.DBPort == 0 {
		return errors.New("DB_PORT is required")
	}
	if c.DBDatabase == "" {
		return errors.New("DB_DATABASE is required")
	}
	if c.DBUser == "" {
		return errors.New("DB_USER is required")
	}
	if c.DBPassword == "" {
		return errors.New("DB_PASSWORD is required")
	}
	if c.Port == 0 {
		return errors.New("PORT is required")
	}
	if c.EncryptionKey == "" {
		return errors.New("ENCRYPTION_KEY is required")
	}
	key, err := base64.StdEncoding.DecodeString(c.EncryptionKey)
	if err != nil {
		return errors.New("ENCRYPTION_KEY must be base64 encoded")
	}
	if len(key) != 32 {
		return errors.New("ENCRYPTION_KEY must be 32 bytes (256-bit)")
	}
	if c.VerifyDomainSecret == "" {
		return errors.New("VERIFY_DOMAIN_SECRET is required")
	}
	return nil
}

func (c *Config) GetEncryptionKey() ([]byte, error) {
	return base64.StdEncoding.DecodeString(c.EncryptionKey)
}

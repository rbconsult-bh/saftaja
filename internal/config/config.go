package config

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Config struct {
	MPGSBaseURL     string `mapstructure:"MPGS_BASE_URL"`
	MPGSMerchantID  string `mapstructure:"MPGS_MERCHANT_ID"`
	MPGSAPIPassword string `mapstructure:"MPGS_API_PASSWORD"`
	DBHost          string `mapstructure:"DB_HOST"`
	DBPort          uint16 `mapstructure:"DB_PORT"`
	DBDatabase      string `mapstructure:"DB_DATABASE"`
	DBUser          string `mapstructure:"DB_USER"`
	DBPassword      string `mapstructure:"DB_PASSWORD"`
}

func LoadConfig(configPath string) (*Config, error) {
	viper.AutomaticEnv()

	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(configPath)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}

		log.Warn().Err(err).Str("config_path", configPath).Msg("no .env file found, using environments variables only")
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

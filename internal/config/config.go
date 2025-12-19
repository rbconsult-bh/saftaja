package config

import (
	"reflect"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Config struct {
	DBHost     string `mapstructure:"DB_HOST"`
	DBPort     uint16 `mapstructure:"DB_PORT"`
	DBDatabase string `mapstructure:"DB_DATABASE"`
	DBUser     string `mapstructure:"DB_USER"`
	DBPassword string `mapstructure:"DB_PASSWORD"`
	Port       int    `mapstructure:"PORT"`
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

	return &cfg, nil
}

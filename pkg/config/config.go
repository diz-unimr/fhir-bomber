package config

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"os"
	"strings"
	"time"
)

type AppConfig struct {
	Bomber Bomber `mapstructure:"bomber"`
	Fhir   Fhir   `mapstructure:"fhir"`
}

type Bomber struct {
	LogLevel string        `mapstructure:"log-level"`
	Interval time.Duration `mapstructure:"interval"`
	Requests string        `mapstructure:"requests"`
	Workers  int           `mapstructure:"workers"`
}

type Http struct {
	Auth Auth   `mapstructure:"auth"`
	Port string `mapstructure:"port"`
}

type Fhir struct {
	Base string `mapstructure:"base"`
	Auth *Auth  `mapstructure:"auth"`
}

type Auth struct {
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
}

func LoadConfig() AppConfig {
	c, err := parseConfig(".")
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to load config file")
		os.Exit(1)
	}

	// log level
	logLevel, err := zerolog.ParseLevel(c.Bomber.LogLevel)
	if err != nil {
		zerolog.SetGlobalLevel(logLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// pretty logging
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	return *c
}

func parseConfig(path string) (config *AppConfig, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("yml")

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(`.`, `_`, `-`, `_`))

	err = viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	err = viper.Unmarshal(&config)
	return config, err
}

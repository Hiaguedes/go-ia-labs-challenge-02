package config

import (
	"errors"
	"io/fs"

	"github.com/spf13/viper"
)

type WeatherConfig struct {
	WeatherApiKey        string `mapstructure:"WEATHER_API_KEY"`
	WebServerWeatherPort string `mapstructure:"WEBSERVER_WEATHER_PORT"`
}

type CepConfig struct {
	WebServerCepPort  string `mapstructure:"WEBSERVER_CEP_PORT"`
	WeatherServiceURL string `mapstructure:"WEATHER_SERVICE_URL"`
}

func LoadWeatherConfig(path string) (*WeatherConfig, error) {
	viper.SetConfigName("app_weather_config")
	viper.SetConfigType("env")
	viper.AddConfigPath(path)
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	// BindEnv only fails when called without key names, so its error is not reachable here.
	_ = viper.BindEnv("WEATHER_API_KEY", "WEBSERVER_WEATHER_PORT")
	if err := viper.ReadInConfig(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	// Read each key directly: Unmarshal only sees keys present in the file,
	// which drops env-only values when the file is absent.
	return &WeatherConfig{
		WeatherApiKey:        viper.GetString("WEATHER_API_KEY"),
		WebServerWeatherPort: viper.GetString("WEBSERVER_WEATHER_PORT"),
	}, nil
}


func LoadCepConfig(path string) (*CepConfig, error) {
	viper.SetConfigName("app_cep_config")
	viper.SetConfigType("env")
	viper.AddConfigPath(path)
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	// BindEnv only fails when called without key names, so its error is not reachable here.
	viper.SetDefault("WEATHER_SERVICE_URL", "http://localhost:8001")
	_ = viper.BindEnv("WEBSERVER_CEP_PORT", "WEATHER_SERVICE_URL")
	if err := viper.ReadInConfig(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	// Read each key directly: Unmarshal only sees keys present in the file,
	// which drops env-only values when the file is absent.
	return &CepConfig{
		WebServerCepPort:  viper.GetString("WEBSERVER_CEP_PORT"),
		WeatherServiceURL: viper.GetString("WEATHER_SERVICE_URL"),
	}, nil
}
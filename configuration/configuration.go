package configuration

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

const appName = "thebeast"

var Config *Configuration

func init() {
	viper.SetDefault("go_env", "development")
	viper.SetDefault("go_port", "8080")
	viper.SetDefault("go_log_level", "info")
	viper.SetDefault("batch_per_core", "20")
	viper.SetDefault("calls_per_host", 5)
	viper.SetDefault("max_retries", 3)
	viper.SetDefault("retry_wait", 5)

	viper.AutomaticEnv()
	viper.SetConfigName("config")                // name of config file (without extension)
	viper.AddConfigPath("/etc/" + appName + "/") // path to look for the config file in
	viper.AddConfigPath("$HOME/." + appName)     // call multiple times to add many search paths
	viper.AddConfigPath(".")                     // optionally look for config in the working directory
	err := viper.ReadInConfig()                  // Find and read the config file
	if err != nil {                              // Handle errors reading the config file
		fmt.Errorf("No config file found: %s \n", err)
	}
	Config = NewConfiguration()
}

// Configuration: settings that are neccesary for server configuration
type Configuration struct {
	AppName      string
	GoEnv        string
	GoPort       string
	GoLogLevel   string
	BatchPerCore int
	CallsPerHost int
	MaxRetries   int
	RetryWait    int
}

func NewConfiguration() *Configuration {
	var config Configuration
	err := config.load()
	if err != nil {
		log.Fatal("Error: couldn't load configuration")
		return nil
	}
	return &config
}

// Global config - thread safe and accessible from all packages

func (c *Configuration) load() error {
	c.AppName = appName
	c.GoEnv = viper.GetString("go_env")
	c.GoLogLevel = viper.GetString("go_log_level")
	c.GoPort = viper.GetString("go_port")
	c.BatchPerCore = viper.GetInt("batch_per_core")
	c.CallsPerHost = viper.GetInt("calls_per_host")
	c.MaxRetries = viper.GetInt("max_retries")
	c.RetryWait = viper.GetInt("retry_wait")

	return nil
}

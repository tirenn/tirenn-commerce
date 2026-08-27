package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/viper"
)

// ServerConfig stores HTTP server configurations
type ServerConfig struct {
	Port        string `mapstructure:"PORT"`
	Environment string `mapstructure:"ENVIRONMENT"`
}

// DatabaseConfig stores MySQL database connection configurations
type DatabaseConfig struct {
	Host     string `mapstructure:"DB_HOST"`
	Port     string `mapstructure:"DB_PORT"`
	User     string `mapstructure:"DB_USER"`
	Password string `mapstructure:"DB_PASSWORD"`
	Name     string `mapstructure:"DB_NAME"`
}

// DSN returns the formatted MySQL Data Source Name
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		d.User,
		d.Password,
		d.Host,
		d.Port,
		d.Name,
	)
}

// JWTConfig stores JSON Web Token parameters
type JWTConfig struct {
	Secret      string `mapstructure:"JWT_SECRET"`
	ExpireHours int    `mapstructure:"JWT_EXPIRE_HOURS"`
	Issuer      string `mapstructure:"JWT_ISSUER"`
}

// Config is the root application configuration holding all environment parameters
type Config struct {
	Server   ServerConfig   `mapstructure:",squash"`
	Database DatabaseConfig `mapstructure:",squash"`
	JWT      JWTConfig      `mapstructure:",squash"`

	// Flat field accessors for convenience
	Port           string
	Environment    string
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	JWTSecret      string
	JWTExpireHours int
}

// LoadConfig initializes Viper, binds environment variables, loads .env if present, and returns Config
func LoadConfig(paths ...string) *Config {
	v := viper.New()

	// 1. Set Defaults
	v.SetDefault("PORT", "8080")
	v.SetDefault("ENVIRONMENT", "development")
	v.SetDefault("DB_HOST", "127.0.0.1")
	v.SetDefault("DB_PORT", "3306")
	v.SetDefault("DB_USER", "root")
	v.SetDefault("DB_PASSWORD", "rootpassword")
	v.SetDefault("DB_NAME", "gocommerce_db")
	v.SetDefault("JWT_SECRET", "super-secret-comic-jwt-key-2026")
	v.SetDefault("JWT_EXPIRE_HOURS", 24)
	v.SetDefault("JWT_ISSUER", "gocommerce-api")

	// 2. Configure Environment Variable Reading
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 3. Configure Config/Env File Reading
	v.SetConfigName(".env")
	v.SetConfigType("env")

	// Search paths for .env
	v.AddConfigPath(".")
	v.AddConfigPath("..")
	v.AddConfigPath("./backend")
	for _, p := range paths {
		v.AddConfigPath(p)
	}

	// Attempt to read config file (if not found, environment variables are used)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("Viper: No .env file found, relying on system environment variables & defaults")
		} else {
			log.Printf("Viper warning: Error reading config file: %v\n", err)
		}
	} else {
		log.Printf("Viper: Loaded configuration from %s\n", v.ConfigFileUsed())
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		log.Printf("Viper: Warning unmarshaling configuration: %v\n", err)
	}

	// Populate flat convenience fields
	cfg.Port = v.GetString("PORT")
	cfg.Environment = v.GetString("ENVIRONMENT")
	cfg.DBHost = v.GetString("DB_HOST")
	cfg.DBPort = v.GetString("DB_PORT")
	cfg.DBUser = v.GetString("DB_USER")
	cfg.DBPassword = v.GetString("DB_PASSWORD")
	cfg.DBName = v.GetString("DB_NAME")
	cfg.JWTSecret = v.GetString("JWT_SECRET")
	cfg.JWTExpireHours = v.GetInt("JWT_EXPIRE_HOURS")

	// Populate nested structs
	cfg.Server.Port = cfg.Port
	cfg.Server.Environment = cfg.Environment
	cfg.Database.Host = cfg.DBHost
	cfg.Database.Port = cfg.DBPort
	cfg.Database.User = cfg.DBUser
	cfg.Database.Password = cfg.DBPassword
	cfg.Database.Name = cfg.DBName
	cfg.JWT.Secret = cfg.JWTSecret
	cfg.JWT.ExpireHours = cfg.JWTExpireHours
	cfg.JWT.Issuer = v.GetString("JWT_ISSUER")

	return &cfg
}

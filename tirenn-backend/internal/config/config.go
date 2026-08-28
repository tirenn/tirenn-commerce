package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/viper"
)

// ServerConfig stores HTTP server configurations
type ServerConfig struct {
	Port         string `mapstructure:"PORT"`
	Environment  string `mapstructure:"ENVIRONMENT"`
	AIServiceURL string `mapstructure:"AI_SERVICE_URL"`
}

// DatabaseConfig stores PostgreSQL database connection configurations
type DatabaseConfig struct {
	Host     string `mapstructure:"DB_HOST"`
	Port     string `mapstructure:"DB_PORT"`
	User     string `mapstructure:"DB_USER"`
	Password string `mapstructure:"DB_PASSWORD"`
	Name     string `mapstructure:"DB_NAME"`
}

// DSN returns the formatted PostgreSQL Data Source Name
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		d.Host,
		d.User,
		d.Password,
		d.Name,
		d.Port,
	)
}

// RedisConfig stores Redis caching and rate-limiting connection parameters
type RedisConfig struct {
	Host                   string `mapstructure:"REDIS_HOST"`
	Port                   string `mapstructure:"REDIS_PORT"`
	Password               string `mapstructure:"REDIS_PASSWORD"`
	DB                     int    `mapstructure:"REDIS_DB"`
	RateLimitReqPerMinute  int    `mapstructure:"RATE_LIMIT_REQUESTS_PER_MINUTE"`
	RateLimitWindowSeconds int    `mapstructure:"RATE_LIMIT_WINDOW_SECONDS"`
}

// Addr returns the formatted Redis host:port address
func (r *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%s", r.Host, r.Port)
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
	Redis    RedisConfig    `mapstructure:",squash"`
	JWT      JWTConfig      `mapstructure:",squash"`

	// Flat field accessors for convenience
	Port                   string
	Environment            string
	AIServiceURL           string
	DBHost                 string
	DBPort                 string
	DBUser                 string
	DBPassword             string
	DBName                 string
	RedisHost              string
	RedisPort              string
	RedisPassword          string
	RedisDB                int
	RateLimitReqPerMinute  int
	RateLimitWindowSeconds int
	JWTSecret              string
	JWTExpireHours         int
	InternalAPIKey         string
}

// LoadConfig initializes Viper, binds environment variables, loads .env if present, and returns Config
func LoadConfig(paths ...string) *Config {
	v := viper.New()

	// 1. Set Defaults for PostgreSQL, Redis, and Services
	v.SetDefault("PORT", "8080")
	v.SetDefault("ENVIRONMENT", "development")
	v.SetDefault("AI_SERVICE_URL", "http://localhost:8000")
	v.SetDefault("INTERNAL_API_KEY", "very-very-secret-internal-key-2026")
	v.SetDefault("DB_HOST", "127.0.0.1")
	v.SetDefault("DB_PORT", "5432")
	v.SetDefault("DB_USER", "gouser")
	v.SetDefault("DB_PASSWORD", "gopassword")
	v.SetDefault("DB_NAME", "gocommerce_db")
	v.SetDefault("REDIS_HOST", "127.0.0.1")
	v.SetDefault("REDIS_PORT", "6379")
	v.SetDefault("REDIS_PASSWORD", "")
	v.SetDefault("REDIS_DB", 0)
	v.SetDefault("RATE_LIMIT_REQUESTS_PER_MINUTE", 120)
	v.SetDefault("RATE_LIMIT_WINDOW_SECONDS", 60)
	v.SetDefault("JWT_SECRET", "super-secret-tirenn-jwt-key-2026")
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
	cfg.AIServiceURL = v.GetString("AI_SERVICE_URL")
	cfg.DBHost = v.GetString("DB_HOST")
	cfg.DBPort = v.GetString("DB_PORT")
	cfg.DBUser = v.GetString("DB_USER")
	cfg.DBPassword = v.GetString("DB_PASSWORD")
	cfg.DBName = v.GetString("DB_NAME")
	cfg.RedisHost = v.GetString("REDIS_HOST")
	cfg.RedisPort = v.GetString("REDIS_PORT")
	cfg.RedisPassword = v.GetString("REDIS_PASSWORD")
	cfg.RedisDB = v.GetInt("REDIS_DB")
	cfg.RateLimitReqPerMinute = v.GetInt("RATE_LIMIT_REQUESTS_PER_MINUTE")
	cfg.RateLimitWindowSeconds = v.GetInt("RATE_LIMIT_WINDOW_SECONDS")
	cfg.JWTSecret = v.GetString("JWT_SECRET")
	cfg.JWTExpireHours = v.GetInt("JWT_EXPIRE_HOURS")
	cfg.InternalAPIKey = v.GetString("INTERNAL_API_KEY")

	// Populate nested structs
	cfg.Server.Port = cfg.Port
	cfg.Server.Environment = cfg.Environment
	cfg.Server.AIServiceURL = cfg.AIServiceURL
	cfg.Database.Host = cfg.DBHost
	cfg.Database.Port = cfg.DBPort
	cfg.Database.User = cfg.DBUser
	cfg.Database.Password = cfg.DBPassword
	cfg.Database.Name = cfg.DBName
	cfg.Redis.Host = cfg.RedisHost
	cfg.Redis.Port = cfg.RedisPort
	cfg.Redis.Password = cfg.RedisPassword
	cfg.Redis.DB = cfg.RedisDB
	cfg.Redis.RateLimitReqPerMinute = cfg.RateLimitReqPerMinute
	cfg.Redis.RateLimitWindowSeconds = cfg.RateLimitWindowSeconds
	cfg.JWT.Secret = cfg.JWTSecret
	cfg.JWT.ExpireHours = cfg.JWTExpireHours
	cfg.JWT.Issuer = v.GetString("JWT_ISSUER")

	return &cfg
}

package main

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"strconv"
)

func loadEnvStr(key string, result *string) {
	s, ok := os.LookupEnv(key)
	if !ok {
		return
	}

	*result = s
}

func loadEnvInt(key string, result *int) {
	s, ok := os.LookupEnv(key)
	if !ok {
		return
	}

	n, err := strconv.Atoi(s)

	if err != nil {
		return
	}

	*result = n
}

func loadEnvUint(key string, result *uint) {
	s, ok := os.LookupEnv(key)
	if !ok {
		return
	}

	n, err := strconv.Atoi(s)

	if err != nil {
		return
	}

	*result = uint(n)
}

type redisConfig struct {
	Host     string `yaml:"host" json:"host"`
	Port     string `yaml:"port" json:"port"`
	Password string `yaml:"password" json:"password"`
	DB       int    `yaml:"db_name" json:"db"`
	SSLMode  string `yaml:"ssl_mode" json:"ssl_mode"`
}

func (p redisConfig) Addr() string {
	return fmt.Sprintf("%s:%s", p.Host, p.Port)
}

func defaultRedisConfig() redisConfig {
	return redisConfig{
		Host:     "localhost",
		Port:     "6379",
		Password: "",
		DB:       0,
		SSLMode:  "disable",
	}
}

func (p *redisConfig) loadFromEnv() {
	loadEnvStr("REDIS_DB_HOST", &p.Host)
	loadEnvStr("REDIS_DB_PORT", &p.Port)
	loadEnvStr("REDIS_DB_PASSWORD", &p.Password)
	loadEnvInt("REDIS_DB", &p.DB)
	loadEnvStr("REDIS_DB_SSL", &p.SSLMode)
}

type ServerConfig struct {
	Host string `yaml:"host" json:"host"`
	Port uint   `yaml:"port" json:"port"`
}

func (l ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", l.Host, l.Port)
}

func defaultServerConfig() ServerConfig {
	return ServerConfig{
		Host: "127.0.0.1",
		Port: 8080,
	}
}

func (l *ServerConfig) loadFromEnv() {
	loadEnvStr("SERVER_HOST", &l.Host)
	loadEnvUint("SERVER_PORT", &l.Port)
}

type config struct {
	Server      ServerConfig `yaml:"server" json:"server"`
	RedisConfig redisConfig  `yaml:"redis" json:"redis"`
}

func (c *config) loadFromEnv() {
	c.Server.loadFromEnv()
	c.RedisConfig.loadFromEnv()
}

func defaultConfig() config {
	return config{
		Server:      defaultServerConfig(),
		RedisConfig: defaultRedisConfig(),
	}
}

func loadConfigFromReader(r io.Reader, c *config) error {
	return yaml.NewDecoder(r).Decode(c)
}

func loadConfigFromFile(fileName string, c *config) error {
	_, err := os.Stat(fileName)
	if err != nil {
		return err
	}

	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	return loadConfigFromReader(f, c)
}

/* How to load the configuration, the highest priority loaded last
 * 1st: -> Initialise to default config
 * 2nd -> Replace with environment variables
 * 3rd -> Replace with configuration file
 */

//func loadConfig(fn string) config {
//	cfg := defaultConfig()
//	cfg.loadFromEnv()
//	loadConfigFromFile(fn, &cfg)
//	return cfg
//}

package config

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env        string     `yaml:"env"         env-default:"production"`
	HTTPServer HTTPServer `yaml:"http_server"`
	Auth       Auth       `yaml:"auth"`
	MQTT       MQTT       `yaml:"mqtt"`
	Storage    Storage    `yaml:"storage"`
	Services   Services   `yaml:"services"`
}

type HTTPServer struct {
	Address         string        `yaml:"address"          env-default:"localhost:8080"`
	Timeout         time.Duration `yaml:"timeout"          env-default:"4s"`
	IdleTimeout     time.Duration `yaml:"idle_timeout"     env-default:"60s"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env-default:"1s"`
}

type Auth struct {
	Oauth2      Oauth2       `yaml:"oauth2"`
	Callback    AuthCallback `yaml:"callback"`
	UserinfoURL string       `yaml:"userinfo_url" env-default:"http://kratos:4444/userinfo"`
}

type AuthCallback struct {
	Host         string        `yaml:"host"          env-default:"127.0.0.1"`
	TTL          time.Duration `yaml:"ttl"           env-default:"5m"`
	ReadTimeout  time.Duration `yaml:"read_timeout"  env-default:"5s"`
	WriteTimeout time.Duration `yaml:"write_timeout" env-default:"5s"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"  env-default:"5s"`
}

type Oauth2 struct {
	ClientID string         `yaml:"client_id" env-default:"smart-pc-cmd"`
	Scopes   []string       `yaml:"scopes"    env-default:"offline,mqtt:pc:state:write,mqtt:pc:command:read,mqtt:pc:log:write,mqtt:pc:status:write"`
	Endpoint Oauth2Endpoint `yaml:"endpoint"`
}

type Oauth2Endpoint struct {
	AuthURL  string `yaml:"auth_url"  env-default:"http://kratos:4444/oauth2/auth"`
	TokenURL string `yaml:"token_url" env-default:"http://kratos:4444/oauth2/token"`
}

type MQTT struct {
	BrokerURL             string `yaml:"broker_url"              env-default:"mqtt://localhost:1883"`
	ClientIDPrefix        string `yaml:"client_id_prefix"        env-default:"smart_pc_agent_"`
	SessionExpiryInterval uint32 `yaml:"session_expiry_interval" env-default:"60"`
	KeepAlive             uint16 `yaml:"keep_alive"              env-default:"20"`
}

type Storage struct {
	Path           string `yaml:"path"            env-default:"./data/storage/db.db"`
	MigrationsPath string `yaml:"migrations_path" env-default:"./data/migrations/sqlite"`
}

type Services struct {
	Pcs Service `yaml:"pcs"`
}

type Service struct {
	Timeout time.Duration `yaml:"timeout"  env-default:"5s"`
	BaseURL string        `yaml:"base_url" env-default:"http://localhost:8080"`
}

func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		log.Fatal("CONFIG_PATH is not set")
	}

	// check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fullConfigPath, _ := filepath.Abs(configPath)
		log.Fatalf("config file does not exists by path \"%s\"", fullConfigPath)
	}

	cfg := &Config{}
	if err := cleanenv.ReadConfig(configPath, cfg); err != nil {
		log.Fatalf("can not read config: %s", err)
	}

	return cfg
}

package config

type Config struct {
	BindAddr string `toml:"bind_addr"`
}

func New() *Config {
	return &Config{
		BindAddr: ":8080",
	}
}

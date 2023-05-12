package main

import (
	"context"
	"flag"
	"log"

	"github.com/BurntSushi/toml"
	"github.com/windnow/tlanalyzer/internal/clickhouse"
	"github.com/windnow/tlanalyzer/internal/config"
	"github.com/windnow/tlanalyzer/internal/tlserver"
)

var (
	configPath string
)

func init() {
	flag.StringVar(&configPath, "config", "config/config.toml", "path to config file")
	flag.Parse()
}

func main() {

	conf := config.New()
	if _, err := toml.DecodeFile(configPath, conf); err != nil {
		log.Printf("Не удалось прочитать конфигурацию из файле %s.", configPath)
	}

	storage, err := clickhouse.New(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	if err := tlserver.Start(conf, storage); err != nil {
		log.Fatal(err)
	}

}

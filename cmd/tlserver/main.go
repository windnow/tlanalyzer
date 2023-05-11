package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/BurntSushi/toml"
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
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	go breakListener(cancel)

	if err := tlserver.Start(ctx, conf); err != nil {
		log.Fatal(err)
	}

}

func breakListener(cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	fmt.Println("Получен сигнал:", sig)
	cancel() // Отменяем контекст
}

package main

import (
	"context"
	//"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/windnow/tlanalyzer/internal/flag"
	"github.com/windnow/tlanalyzer/internal/monitor"
)

var (
	configPath string
	tag        string
	tz         string
	priority   int
	dirs       []string //stringSliceFlag
)

func init() {
	dirs = make([]string, 0)
	flag.StringVar(&configPath, "logcfg", "C:\\Program Files\\1cv8\\conf\\confcfg.xml", "Путь к файлу конфигурации ТЖ")
	flag.IntVar(&priority, "priority", 9, "Приоритет (10-высокий приоритет, 0 - низкий приоритет)")
	flag.StringVar(&tag, "tag", "default", "Тег источника ТЖ")
	flag.StringVar(&tz, "tz", "Asia/Almaty", "Часовой пояс")
	flag.StringSliceVar(&dirs, "dir", "Дополнительный каталог для чтения log файлов")
	flag.Parse()
}

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go breakListener(cancel)
	if priority > 10 {
		priority = 10
	}
	if priority < 0 {
		priority = 0
	}

	monitor, err := monitor.NewMonitor(ctx, dirs, configPath, tz, tag, (10-priority)*10)
	if err != nil {
		log.Fatal(err)
	}

	if err := monitor.Start(); err != nil {
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

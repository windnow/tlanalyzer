package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/windnow/tlanalyzer/internal/myfsm"
)

var (
	configPath string
	dirs       stringSliceFlag
	statusFile string
)

type stringSliceFlag struct {
	values []string
}

func (s stringSliceFlag) String() string {
	return strings.Join(s.values, ", ")
}
func (s *stringSliceFlag) Set(value string) error {
	s.values = append(s.values, value)
	return nil
}

func init() {
	dirs = stringSliceFlag{values: []string{}}
	flag.StringVar(&configPath, "logcfg", "c$\\Program Files\\1cv8\\conf\\conf.cfg", "Путь к файлу конфигурации ТЖ")
	flag.Var(&dirs, "dir", "Дополнительный каталог для чтения log файлов")
	flag.Parse()
}

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go breakListener(cancel)

	monitor := myfsm.NewMonitor(ctx, dirs.values, "C:\\files\\2\\logcfg.xml", "")

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

func main_old() {
	begin := time.Now()
	if len(os.Args) == 1 {
		log.Fatal("directory name not specified")
	}
	rootDir := os.Args[1]

	myfsm.ProcessLogs(rootDir, func(events []myfsm.Event) {
		fmt.Println("ВСЕГО ПРОЧИТАНО", len(events))
	})

	log.Printf("Общее время выполнения: %d", time.Since(begin)/time.Second)

}

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
	dirs       []string //stringSliceFlag
	statusFile string
)

func init() {
	dirs = make([]string, 0)
	flag.StringVar(&configPath, "logcfg", "C:\\Program Files\\1cv8\\conf\\confcfg.xml", "Путь к файлу конфигурации ТЖ")
	flag.StringSliceVar(&dirs, "dir", "Дополнительный каталог для чтения log файлов")
	flag.Parse()
}

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go breakListener(cancel)

	monitor, err := monitor.NewMonitor(ctx, dirs, configPath, "", "")
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

/*func main_old() {
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
*/

package main

import (

	//"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kardianos/service"
	"github.com/windnow/tlanalyzer/internal/common"
	"github.com/windnow/tlanalyzer/internal/flag"
	"github.com/windnow/tlanalyzer/internal/monitor"
	"github.com/windnow/tlanalyzer/internal/program"
)

var (
	mode       string
	operation  string
	configPath string
	tag        string
	tz         string
	priority   int
	dirs       []string //stringSliceFlag
)

func init() {
	dirs = make([]string, 0)
	flag.StringVar(&configPath, "logcfg", "C:\\Program Files\\1cv8\\conf\\logcfg.xml", "Путь к файлу конфигурации ТЖ")
	flag.IntVar(&priority, "priority", 9, "Приоритет (10-высокий приоритет, 0 - низкий приоритет)")
	flag.StringVar(&tag, "tag", "default", "Тег источника ТЖ")
	flag.StringVar(&mode, "mode", "default", "Режим запуска (service-cлужба, default-консоль)")
	flag.StringVar(&operation, "operation", "", "install - установить службу, uninstall - удалить службу")
	flag.StringVar(&tz, "tz", "+06", "Часовой пояс")
	flag.StringSliceVar(&dirs, "dir", "Дополнительный каталог для чтения log файлов")
	flag.Parse()
}

func main() {

	runAsService := mode == "service"

	if priority > 10 {
		priority = 10
	}
	if priority < 0 {
		priority = 0
	}

	m, err := monitor.NewMonitor(dirs, configPath, tz, tag, (10 - priority))
	if err != nil {
		log.Fatal(err)
	}
	var workDir string
	if err := common.WorkingDir(&workDir); err != nil {
		log.Fatal(err)
	}

	if runAsService {

		config := &service.Config{
			Name:             "tlanalyzer",
			DisplayName:      "1C TechLog analyzer",
			Description:      "Парсинг и отправка технологических журналов по HTTP",
			WorkingDirectory: workDir,
			Arguments:        []string{"-mode=service", "-logcfg=\"C:\\Program Files\\1cv8\\conf\\logcfg.xml\"", "-priority=10"},
		}
		p := program.New(m)
		s, err := service.New(p, config)
		if err != nil {
			log.Fatal(err)
		}
		if operation == "install" {
			if err := s.Install(); err != nil {
				log.Fatal("ERROR ON INSTALL ", err.Error())
			}
			log.Println("<-Service installed")
			return
		}
		if operation == "uninstall" {
			if err := s.Uninstall(); err != nil {
				log.Fatal("ERROR ON UNINSTALL ", err.Error())
			}
			log.Println("->Service removed")
			return
		}
		if err := s.Run(); err != nil {
			log.Fatal(err)
		}

	} else {

		go breakListener(m)

		if err := m.Start(nil); err != nil {
			log.Fatal(err)
		}
	}
}

func breakListener(m *monitor.Monitor) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	fmt.Println("Получен сигнал:", sig)
	m.Stop()
}

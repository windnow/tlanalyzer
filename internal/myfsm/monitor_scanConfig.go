package myfsm

import (
	"encoding/xml"
	"os"
	"time"
)

type Log struct {
	Location string `xml:"location,attr"`
	Depth    int    `xml:"history,attr"`
}

type Config struct {
	XMLName xml.Name `xml:"config"`
	Logs    []Log    `xml:"log"`
}

func (m *Monitor) scanConfig() {
	var lastModified time.Time

	m.log.Infof("Старт мониторинга файла конфигурации %s", m.cfg_file)

	defer func() { m.wg.Done() }()

MonitorLoop:
	for {
		select {
		case <-m.ctx.Done():
			break MonitorLoop
		default:
			fileInfo, err := os.Stat(m.cfg_file)
			if err != nil {
				m.log.Errorf("Ошибка чтения информации о файле конфигурации (%s)", err.Error())
				time.Sleep(1 * time.Second)
				continue
			}

			modTime := fileInfo.ModTime()
			if modTime != lastModified {
				lastModified = modTime
				m.checkConfigContent()
			}

			time.Sleep(100 * time.Millisecond)
		}
	}

	m.log.Info("Завершение мониторинга файла конфигурации")

}

func (m *Monitor) checkConfigContent() {
	m.log.Info("Файл конфигурации обновился. Повторное чтение")
	file, err := os.Open(m.cfg_file)
	if err != nil {
		m.log.Errorf("Ошибка открытия файла конфигурации")
	}
	defer file.Close()

	var config Config
	if err := xml.NewDecoder(file).Decode(&config); err != nil {
		m.log.Errorf("Ошибка разбора XML: %s", err.Error())
		return
	}
	for _, l := range config.Logs {
		m.log.Infof("%s -- %d", l.Location, l.Depth)
	}
	m.setFolders(config.Logs)
}

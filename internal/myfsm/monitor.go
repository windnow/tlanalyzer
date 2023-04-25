package myfsm

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

type Monitor struct {
	folders    []string
	statusFile string
	offsets    map[string]int64
	log        *logrus.Logger
}

func NewMonitor(folders []string, statusFile string) (*Monitor, error) {

	return &Monitor{
		folders:    folders,
		statusFile: statusFile,
		log:        logrus.New(),
	}, nil

}

func (m *Monitor) Start() error {
	if len(m.folders) == 0 {
		return errors.New("НЕ УКАЗАНЫ КАТАЛОГИ СКАНИРОВАНИЯ")
	}

	mask := "*.log"
	for {
		for _, folder := range m.folders {
			filepath.Walk(folder, func(path string, fileInfo fs.FileInfo, err error) error {
				if err != nil {
					return err
				}
				matched, err := filepath.Match(mask, fileInfo.Name())
				if err != nil {
					return err
				}

				if !fileInfo.IsDir() && matched {
					info := fmt.Sprintf("Чтение %s", path)
					offset, ok := m.offsets[path]
					if !ok {
						offset = 0
					}
					start := time.Now()
					offset, events, err := m.ScanFile(offset, path)
					if err != nil {
						duration := time.Since(start)
						m.log.Errorf("%s. Ошибка. %s. Длительность: %d мк.с.", info, err.Error(), duration/time.Microsecond)
						return nil
					}

					err = m.SendEvents(events)
					if err != nil {
						return nil
					}
					m.offsets[path] = offset

					m.log.Info(info)
				}
				return nil
			})

			time.Sleep(10 * time.Second)
		}
	}

}

func (m *Monitor) ScanFile(offset int64, filePath string) (int64, []*Event, error) {

	return 0, nil, nil
}

func (m *Monitor) SendEvents(events []*Event) error {

	return errors.New("ЗАГЛУШКА")

}

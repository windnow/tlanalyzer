package myfsm

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type Monitor struct {
	mutex       sync.Mutex
	cfg_folders []string
	cfg_file    string
	folders     []string
	statusFile  string
	offsets     map[string]int64
	log         *logrus.Logger
}

func NewMonitor(folders []string, cfg_file, statusFile string) *Monitor {

	monitor := &Monitor{
		folders:    folders,
		cfg_file:   cfg_file,
		statusFile: statusFile,
	}
	monitor.log = monitor.loggerConfig()
	return monitor
}

func (m *Monitor) scanFolders() error {

	mask := "*.log"
	for {

		folders := m.getFolders()

		for _, folder := range folders {
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
						duration := time.Since(start)
						m.log.Errorf("%s. Ошибка при отправке (%s). Длительность %d мк.с.", info, err.Error(), duration/time.Microsecond)
						return nil
					}
					m.offsets[path] = offset

					m.log.Info(info)
				}
				return nil
			})

			time.Sleep(10 * time.Second)
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func (m *Monitor) ScanFile(offset int64, filePath string) (int64, []Event, error) {

	fileName := filepath.Base(filePath)
	fileNameWithoutExt := "20" + strings.TrimSuffix(fileName, filepath.Ext(fileName))

	file, err := os.Open(filePath)
	if err != nil {
		return 0, nil, err
	}

	fsm := myFSM{fileName: fileNameWithoutExt}

	defer func() {
		file.Close()
	}()

	scanner := bufio.NewScanner(file)
	scanner.Split(scanLinesWithoutBOM)

	scanner.Buffer(make([]byte, 0), 1024*1024)
	reNewEvent := regexp.MustCompile(`^\d\d:\d\d\.\d{6}`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if reNewEvent.MatchString(line) {
			if fsm.Event != nil {
				fsm.Event = fsm.FinalizeEvent
			} else {
				fsm.Event = fsm.NewEvent
			}
		} else {
			line = "\n" + line
		}

		for _, c := range line {
			fsm.Update(c)
		}

	}
	a, err := file.Seek(0, os.SEEK_CUR)
	if err != nil {
		return 0, nil, errors.New("НЕ УДАЛОСЬ ПОЛУЧИТЬ СМЕЩЕНИЕ ПОСЛЕ ЗАВЕРШЕНИЯ ЧТЕНИЯ")
	}

	if err := scanner.Err(); err != nil {
		return 0, nil, err
	}
	if fsm.Event != nil {
		fsm.Event = fsm.FinalizeEvent
		fsm.Update(0)
	}

	return a, fsm.events, nil
}

func (m *Monitor) SendEvents(events []Event) error {

	return errors.New("ЗАГЛУШКА")
}

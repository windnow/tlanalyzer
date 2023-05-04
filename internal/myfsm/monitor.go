//go:build windows

package myfsm

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
	"github.com/windnow/tlanalyzer/internal/common"
)

type Monitor struct {
	ctx         context.Context
	wg          sync.WaitGroup
	mutex       sync.Mutex
	cfg_folders []Log
	cfg_file    string
	folders     []Log
	statusFile  string
	log         *logrus.Logger
}

func NewMonitor(ctx context.Context, folders []string, cfg_file, statusFile string) (monitor *Monitor) {

	monitor = &Monitor{
		ctx: ctx,
		folders: func() []Log {
			logFolders := make([]Log, 0, len(folders))

			for _, folder := range folders {
				logFolders = append(logFolders, Log{Location: folder, Depth: -1})
			}
			return logFolders
		}(),
		log: func() *logrus.Logger {
			return &logrus.Logger{
				Out:   os.Stderr,
				Level: logrus.DebugLevel,
				Formatter: &easy.Formatter{
					TimestampFormat: "2006-01-02 15:04:05",
					LogFormat:       "[%lvl%]: %time% - %msg%\n",
				},
			}
		}(),
		cfg_file:   cfg_file,
		statusFile: statusFile,
	}
	return
}

func (m *Monitor) Start() error {

	m.wg.Add(1)
	go m.scanConfig()

	m.wg.Add(1)
	go m.scanFolder()

	m.wg.Wait()
	return nil
}

var mask = "*.log"

func (m *Monitor) scanFolder() {
	defer m.wg.Done()

FoldersScanner:
	for {

		folders := m.getFolders()

		for _, folder := range folders {

			select {
			case <-m.ctx.Done():
				m.log.Errorf("СКАНИРОВАНИЕ \"%s\" ПРЕРВАНО ПО СИГНАЛУ", folder.Location)
				break FoldersScanner
			default:
				filepath.Walk(folder.Location, m.pathChecker)
			}

			time.Sleep(10 * time.Second)
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func (m *Monitor) SendEvents(events []Event) error {

	return errors.New("ЗАГЛУШКА")
}

func (m *Monitor) pathChecker(path string, fileInfo fs.FileInfo, err error) error {
	if err != nil {
		return err
	}
	matched, err := filepath.Match(mask, fileInfo.Name())
	if err != nil {
		return err
	}

	if !fileInfo.IsDir() && matched {
		canBeDeleted := common.FileCanBeDeleted(path)
		info := fmt.Sprintf("Чтение \"%s\" (%t)", path, canBeDeleted)
		start := time.Now()
		events, err := m.ScanFile(m.ctx, path)
		if err != nil {
			duration := time.Since(start)
			m.log.Errorf("%s. Ошибка. %s. Длительность: %d мк.с.", info, err.Error(), duration/time.Microsecond)
			return err
		}

		eventsCount := len(events)
		if eventsCount > 0 {
			err = m.SendEvents(events)
			if err != nil {
				duration := time.Since(start)
				m.log.Errorf("%s. Ошибка при отправке (%s). Длительность %d мк.с., число событий: %d", info, err.Error(), duration/time.Microsecond, len(events))
				return nil
			}
			m.log.Info(info)
		}

	}
	return nil
}

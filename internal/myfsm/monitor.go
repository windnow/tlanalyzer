//go:build windows

package myfsm

import (
	"context"
	"encoding/json"
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
	timezone    string

	folders    []Log
	statusFile string
	log        *logrus.Logger
}

func NewMonitor(ctx context.Context, folders []string, cfg_file, statusFile, timezone string) (monitor *Monitor) {
	logFolders := make([]Log, 0, len(folders))

	for _, folder := range folders {
		logFolders = append(logFolders, Log{Location: folder, Depth: -1})
	}

	log := &logrus.Logger{
		Out:   os.Stderr,
		Level: logrus.DebugLevel,
		Formatter: &easy.Formatter{
			TimestampFormat: "2006-01-02 15:04:05",
			LogFormat:       "[%lvl%]: %time% - %msg%\n",
		},
	}
	tz := "Asia/Almaty"
	if timezone != "" {
		tz = timezone
	}

	return &Monitor{
		ctx:        ctx,
		folders:    logFolders,
		log:        log,
		cfg_file:   cfg_file,
		statusFile: statusFile,
		timezone:   tz,
	}
}

func (m *Monitor) Start() error {

	m.wg.Add(1)
	go m.scanConfig()

	m.wg.Add(1)
	go m.scanFolder()

	m.wg.Wait()
	m.log.Info("Все процессы завершены")
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
				time.Sleep(500 * time.Millisecond)
			}

		}

		time.Sleep(100 * time.Millisecond)
	}
}

func (m *Monitor) SendEvents(events []Event) error {
	if len(events) == 0 {
		return nil
	}

	loc, err := time.LoadLocation("Asia/Almaty")
	if err != nil {
		return err
	}
	eventsJSON, err := json.Marshal(events)
	if err != nil {
		return err
	}
	m.log.Info(string(eventsJSON))
	for _, event := range events {
		t, _ := time.ParseInLocation("20060102 15:04:05.000000", *event.GetField("time"), loc)
		m.log.Info("================ ", *event.GetField("time"), "->", t)
	}

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
		select {
		case <-m.ctx.Done():
			errMsg := fmt.Sprintf("СКАНИРОВАНИЕ ДИРРЕКТОРИИ \"%s\" ПРЕРВАНО ПО СИГНАЛУ", path)
			m.log.Error(errMsg)
			return errors.New(errMsg)
		default:
			canBeDeleted := common.FileCanBeDeleted(path)
			info := fmt.Sprintf("Чтение \"%s\" (%t)", path, canBeDeleted)
			start := time.Now()
			events, err := m.ScanFile(path)
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
					m.log.Errorf("%s. Ошибка при отправке (%s). Длительность %d мк.с., число событий: %d", info, err.Error(), duration/time.Microsecond, eventsCount)
					return nil
				}
				m.log.Info(info)
			}

		}
	}
	return nil
}

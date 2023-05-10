package monitor

import (
	"bufio"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
	"github.com/windnow/tlanalyzer/internal/common"
	"github.com/windnow/tlanalyzer/internal/internalprocessor"
	"github.com/windnow/tlanalyzer/internal/myfsm"
	"github.com/windnow/tlanalyzer/internal/processor"
	// "github.com/windnow/tlanalyzer/internal/redisprocessor"
)

var mask = "*.log"

type Monitor struct {
	ctx         context.Context
	wg          sync.WaitGroup
	mutex       sync.Mutex
	cfg_folders []Log
	cfg_file    string
	location    *time.Location

	folders   []Log
	log       *logrus.Logger
	processor processor.Processor
}

func NewMonitor(ctx context.Context, folders []string, cfg_file, timezone string) (monitor *Monitor, err error) {
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

	loc, err := time.LoadLocation(tz)
	if err != nil {

		return nil, err

	}

	// processor, err := redisprocessor.NewProcessor(ctx, log)
	processor, err := internalprocessor.NewProcessor(ctx, log)
	if err != nil {
		return nil, err
	}

	return &Monitor{
		ctx:       ctx,
		folders:   logFolders,
		log:       log,
		cfg_file:  cfg_file,
		location:  loc,
		processor: processor,
	}, nil
}

func (m *Monitor) ScanFile(filePath string) ([]myfsm.Event, error) {

	fileName := filepath.Base(filePath)
	fileNameWithoutExt := "20" + strings.TrimSuffix(fileName, filepath.Ext(fileName))

	fsm := myfsm.NewFSM(fileNameWithoutExt)
	file, scanner, err := getFileScanner(filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		file.Close()
	}()

	for scanner.Scan() {
		select {
		case <-m.ctx.Done():
			errMsg := fmt.Sprintf("СКАНИРОВАНИЕ ФАЙЛА \"%s\" ПРЕРВАНО (ПО СИГНАЛУ)", fileName)
			return nil, errors.New(errMsg)
		default:
			fsm.ProcessLine(scanner.Text())
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if fsm.Event != nil {
		fsm.Event = fsm.FinalizeEvent
		fsm.Update(0)
	}

	events := fsm.GetEvents()
	var i = 0
	for _, event := range events {
		event.SetIndex(i)
		event.ParseTime(m.location)
		i++
	}

	return events, nil
}

func getFileScanner(filePath string) (file *os.File, scanner *bufio.Scanner, err error) {
	file, err = os.Open(filePath)
	if err != nil {
		return nil, nil, err
	}

	scanner = bufio.NewScanner(file)
	scanner.Split(scanLinesWithoutBOM)

	scanner.Buffer(make([]byte, 0), 1024*1024)
	return
}

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

func (m *Monitor) Start() error {

	defer func() {
		m.processor.Close()
	}()

	m.wg.Add(1)
	go m.scanConfig()

	m.wg.Add(1)
	go m.scanFolder()

	m.wg.Wait()
	m.log.Info("Все процессы завершены")
	return nil
}

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
				err = m.processor.SendEvents(events)
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

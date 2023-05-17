package monitor

import (
	"bufio"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
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

type LogOfset struct {
	Offset    int64
	LastEvent time.Time
	Idx       int
	Depth     int
}

type Monitor struct {
	ctx         context.Context
	wg          sync.WaitGroup
	mutex       sync.Mutex
	cfg_folders []Log
	cfg_file    string
	location    *time.Location
	tag         string
	priority    int

	LogOfsets map[string]LogOfset
	folders   []Log
	log       *logrus.Logger
	processor processor.Processor
}

func NewMonitor(ctx context.Context, folders []string, cfg_file, timezone, tag string, priority int) (*Monitor, error) {
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

	loc, err := time.LoadLocation(timezone)
	if err != nil {

		return nil, err

	}
	monitor := &Monitor{
		ctx:       ctx,
		folders:   logFolders,
		log:       log,
		cfg_file:  cfg_file,
		location:  loc,
		tag:       tag,
		LogOfsets: make(map[string]LogOfset),
		priority:  priority,
		wg:        sync.WaitGroup{},
	}

	// processor, err := redisprocessor.NewProcessor(ctx, log)
	processor, err := internalprocessor.NewProcessor(ctx, log, &monitor.wg)
	if err != nil {
		return nil, err
	}
	monitor.processor = processor
	return monitor, nil
}

func (m *Monitor) ScanFile(filePath string) (events []myfsm.Event, offset int64, i int, err error) {

	fileName := filepath.Base(filePath)
	fileNameWithoutExt := "20" + strings.TrimSuffix(fileName, filepath.Ext(fileName))

	fsm := myfsm.NewFSM(fileNameWithoutExt)
	file, scanner, err := getFileScanner(filePath)
	if offset, i = m.getOffset(filePath); offset > 0 {
		file.Seek(offset, io.SeekStart)
	}
	if err != nil {
		return nil, 0, 0, err
	}
	defer func() {
		file.Close()
	}()

	for scanner.Scan() {
		select {
		case <-m.ctx.Done():
			errMsg := fmt.Sprintf("СКАНИРОВАНИЕ ФАЙЛА \"%s\" ПРЕРВАНО (ПО СИГНАЛУ)", fileName)
			return nil, 0, 0, errors.New(errMsg)
		default:
			if m.priority > 0 {
				time.Sleep(time.Duration(m.priority) * time.Millisecond)
			}
			fsm.ProcessLine(scanner.Text())
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, 0, 0, err
	}
	if fsm.Event != nil {
		fsm.Event = fsm.FinalizeEvent
		fsm.Update(0)
	}

	offset, err = file.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, 0, 0, err
	}

	events = fsm.GetEvents()
	for _, event := range events {
		if err := event.ParseTime(m.location); err != nil {
			m.log.Warnf("Не удалось преобразовать в дату строку вида \"%s\"", *event.GetField("time"))
		}

		event.SetIndex(i)
		event.SetTag(m.tag)
		event.ParsePath(filePath)
		i++
	}

	return events, offset, i, nil
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

	m.restoreLogOffsets()

	defer func() {
		m.processor.Close()
	}()

	if common.FileExistsAndIsReadable(m.cfg_file) {
		m.wg.Add(1)
		go m.scanConfig()
	} else {
		if m.cfg_file != "" {
			m.log.Warn("НЕ ВЕРНО ЗАДАН ФАЙЛ КОНФИГУРАЦИИ")
		}
	}

	m.wg.Add(1)
	go m.scanFolder()

	m.wg.Wait()
	m.log.Info("Все процессы завершены")
	return nil
}
func (m *Monitor) getOffset(path string) (int64, int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	LogOfset, ok := m.LogOfsets[path]
	if ok {
		return LogOfset.Offset, LogOfset.Idx
	}

	return 0, 0
}

func (m *Monitor) restoreLogOffsets() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	b, err := ioutil.ReadFile("offsets")
	if err != nil {
		return
	}
	json.Unmarshal(b, &m.LogOfsets)

}

func (m *Monitor) LogOffset(e myfsm.Event, path string, offset int64, i int, depth int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	switch be := e.(type) {
	case *myfsm.BulkEvent:
		m.LogOfsets[path] = LogOfset{
			Offset:    offset,
			LastEvent: be.Time,
			Idx:       i,
			Depth:     depth,
		}
		jsonData, err := json.Marshal(m.LogOfsets)
		if err != nil {
			m.log.Warnf("Не удалось сериализировать данные смещений (%s)", err.Error())
			return
		}
		err = ioutil.WriteFile("offsets", jsonData, 0644)
		if err != nil {

			m.log.Warnf("Не удалось сохранить данные смещений (%s)", err.Error())

		}
	}
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
				filepath.Walk(folder.Location, func(path string, fileInfo fs.FileInfo, err error) error {
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
							info := fmt.Sprintf("Чтение \"%s\"", path)
							start := time.Now()
							events, offset, i, err := m.ScanFile(path)
							if err != nil {
								duration := time.Since(start)
								m.log.Errorf("%s. Ошибка. %s. Длительность: %d мк.с.", info, err.Error(), duration/time.Microsecond)
								return err
							}
							eventsCount := len(events)
							duration := time.Since(start)
							info = fmt.Sprintf("%s: Прочитано %d за %d мк.с.", info, eventsCount, duration/time.Microsecond)
							if eventsCount > 0 {
								start = time.Now()
								info = fmt.Sprintf("%s: Отправка событий", info)
								err = m.processor.SendEvents(events)
								if err != nil {
									duration := time.Since(start)
									m.log.Errorf("%s. Ошибка при отправке (%s). Длительность %d мк.с.", info, err.Error(), duration/time.Microsecond)
									return nil
								}

								m.LogOffset(events[len(events)-1], path, offset, i, folder.Depth)

								duration := time.Since(start)
								info = fmt.Sprintf("%s: Отправлено за %d мк.с.", info, duration/time.Microsecond)
								m.log.Info(info)
							}
						}
					}
					return nil
				})
			}

		}

	}
}

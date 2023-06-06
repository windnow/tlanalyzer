package monitor

import (
	"bufio"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kardianos/service"
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
	Modified  time.Time
	Idx       int
	Depth     int
	Tail      []byte
}

type Monitor struct {
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	mutex       sync.Mutex
	cfg_folders []Log
	cfg_file    string
	location    *time.Location
	tag         string
	priority    int

	workDir   string
	LogOfsets map[string]LogOfset
	folders   []Log
	log       *logrus.Logger
	logFile   *os.File
	processor processor.Processor
	service   service.Service
}

type Log struct {
	Location string `xml:"location,attr"`
	Depth    int    `xml:"history,attr"`
}

type Config struct {
	XMLName xml.Name `xml:"config"`
	Logs    []Log    `xml:"log"`
}

func NewMonitor(folders []string, cfg_file, timezone, tag string, priority int) (*Monitor, error) {
	if priority < 0 {
		priority = 0
	}
	logFolders := make([]Log, 0, len(folders))

	for _, folder := range folders {
		logFolders = append(logFolders, Log{Location: folder, Depth: -1})
	}
	var workDir string
	if err := common.WorkingDir(&workDir); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(fmt.Sprintf("%s/tlanalyzer.log", workDir), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	log := &logrus.Logger{
		Out:   file,
		Level: logrus.DebugLevel,
		Formatter: &easy.Formatter{
			TimestampFormat: "2006-01-02 15:04:05",
			LogFormat:       "[%lvl%]: %time% - %msg%\n",
		},
	}

	offset, err := strconv.Atoi(timezone)
	if err != nil {
		return nil, err
	}

	offset = offset * 60 * 60
	loc := time.FixedZone(timezone, offset)

	ctx, cancel := context.WithCancel(context.Background())
	monitor := &Monitor{
		ctx:       ctx,
		cancel:    cancel,
		folders:   logFolders,
		log:       log,
		logFile:   file,
		cfg_file:  cfg_file,
		location:  loc,
		tag:       tag,
		LogOfsets: make(map[string]LogOfset),
		priority:  priority,
		wg:        sync.WaitGroup{},
		workDir:   workDir,
	}

	// processor, err := redisprocessor.NewProcessor(ctx, log)
	processor, err := internalprocessor.NewProcessor(ctx, log, &monitor.wg)
	if err != nil {
		return nil, err
	}
	monitor.processor = processor
	return monitor, nil
}

func (m *Monitor) ScanFile(filePath string) (events []myfsm.Event, offset int64, i int, tail string, modified time.Time, err error) {

	fileName := filepath.Base(filePath)
	fileNameWithoutExt := "20" + strings.TrimSuffix(fileName, filepath.Ext(fileName))

	fsm := myfsm.NewFSM(fileNameWithoutExt)
	offset, i, tail, modified = m.getOffset(filePath)
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, 0, 0, "", modified, err
	}
	if fileInfo.ModTime() == modified {
		return
	}
	file, scanner, err := getFileScanner(filePath)
	if err != nil {
		return nil, 0, 0, "", modified, err
	}
	defer file.Close()
	if offset > 0 {
		for j := int64(0); j < offset; j++ {
			scanner.Scan()
		}
	}
	if len(tail) > 0 {
		fsm.ProcessLine(tail)
	}
FileScan:
	for scanner.Scan() {
		offset++
		select {
		case <-m.ctx.Done():
			errMsg := fmt.Sprintf("СКАНИРОВАНИЕ ФАЙЛА \"%s\" ПРЕРВАНО (ПО СИГНАЛУ)", fileName)
			return nil, 0, 0, "", modified, errors.New(errMsg)
		default:
			if m.priority > 0 {
				time.Sleep(time.Duration(m.priority) * time.Millisecond)
			}
			fsm.ProcessLine(scanner.Text())
			if fsm.Full() {
				break FileScan
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, 0, 0, "", modified, err
	}
	if fsm.Event != nil && !fsm.Full() {
		fsm.Event = fsm.FinalizeEvent
		fsm.Update(0)
		modified = fileInfo.ModTime()
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

	return events, offset, i, fsm.Tail(), modified, nil
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

func (m *Monitor) Start(s service.Service) error {

	if s != nil {
		m.service = s
	}

	m.restoreLogOffsets()

	defer func() {
		m.processor.Close()
	}()

	if common.FileExistsAndIsReadable(m.cfg_file) {
		m.wg.Add(1)
		go m.scanConfig()
	} else {
		if m.cfg_file != "" {
			m.log.Warn(fmt.Sprintf("НЕ ВЕРНО ЗАДАН ФАЙЛ КОНФИГУРАЦИИ (%s)", m.cfg_file))
		}
	}

	m.wg.Add(1)
	go m.scanFolder()

	m.wg.Add(1)
	go m.monitorOfsets()

	m.wg.Wait()
	m.logFile.Close()
	m.log.Info("Все процессы завершены")
	return nil
}

func (m *Monitor) Stop() error {
	m.cancel()
	if m.service != nil {
		return m.service.Stop()
	}
	return nil
}

func (m *Monitor) getOffset(path string) (offset int64, idx int, tail string, lastModified time.Time) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	LogOfset, ok := m.LogOfsets[path]
	if ok {
		return LogOfset.Offset, LogOfset.Idx, string(LogOfset.Tail), LogOfset.Modified
	}

	return 0, 0, "", LogOfset.Modified
}

func (m *Monitor) restoreLogOffsets() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	b, err := os.ReadFile(fmt.Sprintf("%s\\offsets", m.workDir))
	if err != nil {
		return
	}
	json.Unmarshal(b, &m.LogOfsets)
}

func (m *Monitor) monitorOfsets() {
	defer m.wg.Done()
	wg := sync.WaitGroup{}
	ticker := time.NewTicker(5 * time.Second)
	tick := 0
INFINITIE:
	for {
		select {
		case <-m.ctx.Done():
			break INFINITIE
		case <-ticker.C:
			tick++
			if tick%720 == 0 {
				tick = 0
				wg.Add(1)
				go func() {
					defer wg.Done()
					m.mutex.Lock()
					defer m.mutex.Unlock()
					notFounds := make([]string, 0)
					for key, _ := range m.LogOfsets {
						if _, err := os.Stat(key); os.IsNotExist(err) {
							notFounds = append(notFounds, key)
						}
					}
					for _, key := range notFounds {
						delete(m.LogOfsets, key)
						m.log.Infof("Path \"%s\" removed from offsets list", key)
					}
				}()

				wg.Wait()
			}
		}
	}
}

func (m *Monitor) LogOffset(e myfsm.Event, path string, offset int64, i int, tail string, modified time.Time, depth int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	switch be := e.(type) {
	case *myfsm.BulkEvent:
		m.LogOfsets[path] = LogOfset{
			Offset:    offset,
			LastEvent: be.Time,
			Idx:       i,
			Tail:      []byte(tail),
			Modified:  modified,
			Depth:     depth,
		}
		jsonData, err := json.Marshal(m.LogOfsets)
		if err != nil {
			m.log.Warnf("Не удалось сериализировать данные смещений (%s)", err.Error())
			return
		}
		err = os.WriteFile("offsets", jsonData, 0644)
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
		/*if len(folders) == 0 {
			m.log.Warn("Список каталогов пуст. Выходим")
			m.Stop()
			return
		}
		*/

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
							events, offset, i, tail, modified, err := m.ScanFile(path)
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

								m.LogOffset(events[len(events)-1], path, offset, i, tail, modified, folder.Depth)

								duration := time.Since(start)
								info = fmt.Sprintf("%s: Отправлено за %d мк.с.", info, duration/time.Microsecond)
								m.log.Info(info)
							}
						}
					}
					time.Sleep(10 * time.Millisecond)
					return nil
				})
			}

		}

	}
}

func (m *Monitor) scanSQL() {

}

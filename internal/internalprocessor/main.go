package internalprocessor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/windnow/tlanalyzer/internal/common"
	"github.com/windnow/tlanalyzer/internal/myfsm"
)

type Poster func(io.Reader) error

type Config struct {
	SendThreshold  int    `json:"send threshold"`
	Limit          int    `json:"limit"`
	MaxInterval    int    `json:"max interval"`
	ServerEndpoint string `json:"server endpoint"`
	CacheFileName  string `json:"cache file name"`
}

type InternalProcessor struct {
	lastSend time.Time
	wg       *sync.WaitGroup
	events   []myfsm.Event
	mutex    sync.Mutex
	post     Poster
	log      *logrus.Logger
	ctx      context.Context
	config   Config
	lastMsg  string
}

func NewProcessor(ctx context.Context, log *logrus.Logger, wg *sync.WaitGroup) (*InternalProcessor, error) {
	processor := &InternalProcessor{
		wg:       wg,
		ctx:      ctx,
		log:      log,
		lastSend: time.Now(),
		events:   make([]myfsm.Event, 0),
	}
	processor.post = processor.localPoster

	processor.loadConfig()
	processor.restore()
	processor.log.Infof(
		"Параметры передачи:\n\tРазмер порции передачи: %d;\n\tМаксимальное количество событий: %d; \n\tМаксимальный интервал между отправками (сек.): %d",
		processor.config.SendThreshold,
		processor.config.Limit,
		processor.config.MaxInterval,
	)
	processor.wg.Add(1)
	go processor.startMonitoring()

	return processor, nil
}

func (p *InternalProcessor) loadConfig() {

	config := Config{}
	data, err := os.ReadFile("int_config.json")
	if err != nil {
		p.log.Warn("Не удалось прочитать файл конфигурации. Размеры данных установлены по умолчанию")
	} else {
		err = json.Unmarshal(data, &config)
		if err != nil {
			p.log.Warn("Не удалось разобрать файл конфигурации. Размеры данных установлены по умолчанию")
		}
	}

	if config.SendThreshold == 0 {
		config.SendThreshold = 15000
	}
	if config.Limit == 0 {
		config.Limit = 250000
	}
	if config.MaxInterval == 0 {
		config.MaxInterval = 300
	}

	if len(config.CacheFileName) == 0 {
		config.CacheFileName = "cache.json"
	}

	if len(config.ServerEndpoint) == 0 {
		config.ServerEndpoint = "http://192.168.24.110:8080/set"
	}

	if config.SendThreshold > config.Limit {
		config.Limit, config.SendThreshold = config.SendThreshold, config.Limit
	}
	p.config = config
}

func (p *InternalProcessor) startMonitoring() {
	defer p.wg.Done()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			ticker.Stop()
			p.log.Info("Процесс мониторинга (внутренний) завершен по сигналу")
			return
		case <-ticker.C:
			p.SendDataIfThresholdReached()
		}
	}

}

func (p *InternalProcessor) localPoster(body io.Reader) error {
	r, err := http.NewRequestWithContext(p.ctx, http.MethodPost, p.config.ServerEndpoint, body)
	if err != nil {
		return err
	}

	r.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return errors.New(string(body))
	}

	return nil

}

func (p *InternalProcessor) send(events []myfsm.Event) error {

	var compressed bytes.Buffer
	if err := common.Compress(&compressed, events); err != nil {
		return err
	}
	if err := p.post(&compressed); err != nil {
		return err
	}

	return nil
}

func (p *InternalProcessor) timeout() bool {
	duration := time.Since(p.lastSend)
	return duration > time.Duration(p.config.MaxInterval)*time.Second
}

func (p *InternalProcessor) SendDataIfThresholdReached() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	cap := len(p.events)
	msg := fmt.Sprintf("check capacity: %d", cap)
	if p.lastMsg != msg {
		p.log.Info(msg)
		p.lastMsg = msg
	}
	if cap > p.config.SendThreshold || (p.timeout() && cap > 0) {
		portion := p.config.Limit
		for cap > 0 {

			border := cap
			if border > portion {
				border = portion
			}

			if err := p.send(p.events[:border]); err != nil {
				p.log.Errorf("Ошибка отправки событий: %s", err.Error())
				jsonData, _ := json.Marshal(p.events)
				os.WriteFile(p.config.CacheFileName, jsonData, 0644)
				return
			}

			p.events = p.events[border:]

			cap = len(p.events)
		}
		p.lastSend = time.Now()
		jsonData, _ := json.Marshal(p.events)
		os.WriteFile(p.config.CacheFileName, jsonData, 0644)
	}
}

func (p *InternalProcessor) restore() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	b, err := os.ReadFile(p.config.CacheFileName)
	if err != nil {
		p.log.Info("Файл статусов обработки не найден. Пропущено")
		return
	}

	var events []*myfsm.BulkEvent
	if err := json.Unmarshal(b, &events); err != nil {
		p.log.Info("Не удалось разобрать файл статусов обработки. Пропущено")
		return
	}
	p.events = make([]myfsm.Event, len(events))

	for i, v := range events {
		p.events[i] = v
	}
	p.log.Info("После перезапуска восстановлено событий: ", len(p.events))
}

func (p *InternalProcessor) save() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	jsonData, err := json.Marshal(p.events)
	if err != nil {
		return err
	}

	return os.WriteFile(p.config.CacheFileName, jsonData, 0644)

}

func (p *InternalProcessor) capacity() int {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return len(p.events)

}
func (p *InternalProcessor) append(events []myfsm.Event) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.events = append(p.events, events...)
}

func (p *InternalProcessor) Close() {
	p.log.Info("Завершение работы внутреннего процессора")
}

func (p *InternalProcessor) SendEvents(events []myfsm.Event) error {
	if p.capacity() > 50000 {
		return errors.New("ДОСТИГНУТ ЛИМИТ ХРАНИЛИЩА. ПРОВЕРЬТЕ ПЕРЕДАЧУ")
	}
	p.append(events)
	return p.save()
}

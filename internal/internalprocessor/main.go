package internalprocessor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/windnow/tlanalyzer/internal/myfsm"
)

type Config struct {
	SendThreshold int `json:"socket"`
	Limit         int `json:"password"`
}

type InternalProcessor struct {
	wg        *sync.WaitGroup
	cacheFile string
	events    []myfsm.Event
	mutex     sync.Mutex
	log       *logrus.Logger
	ctx       context.Context
	config    Config
	lastMsg   string
}

func NewProcessor(ctx context.Context, log *logrus.Logger, wg *sync.WaitGroup) (*InternalProcessor, error) {
	config := Config{}
	data, err := ioutil.ReadFile("int_config.json")
	if err != nil {
		log.Warn("Не удалось прочитать файл конфигурации. Размеры данных установлены по умолчанию")
	} else {
		err = json.Unmarshal(data, &config)
		if err != nil {
			log.Warn("Не удалось разобрать файл конфигурации. Размеры данных установлены по умолчанию")
		}
	}
	if config.SendThreshold == 0 {
		config.SendThreshold = 5000
	}
	if config.Limit == 0 {
		config.SendThreshold = 50000
	}
	if config.SendThreshold > config.Limit {
		config.Limit, config.SendThreshold = config.SendThreshold, config.Limit
	}
	processor := &InternalProcessor{
		cacheFile: ".\\cache.out",
		wg:        wg,
		ctx:       ctx,
		log:       log,
		events:    make([]myfsm.Event, 0),
		config:    config,
	}

	processor.restore()
	processor.wg.Add(1)
	go processor.startMonitoring()

	return processor, nil
}

func (p *InternalProcessor) startMonitoring() {
	defer p.wg.Done()

	ticker := time.NewTicker(500 * time.Millisecond)
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

func (p *InternalProcessor) SendDataIfThresholdReached() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	cap := len(p.events)
	msg := fmt.Sprintf("check capacity: %d", cap)
	if p.lastMsg != msg {
		p.log.Info(msg)
		p.lastMsg = msg
	}
	if cap > 5000 {
		p.events = make([]myfsm.Event, 0)
		cap := len(p.events)
		p.log.Info("reset slice: ", cap)
	}
}

func (p *InternalProcessor) restore() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	b, err := ioutil.ReadFile(p.cacheFile)
	if err != nil {
		return
	}

	var events []*myfsm.BulkEvent
	if err := json.Unmarshal(b, &events); err != nil {
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

	return ioutil.WriteFile(p.cacheFile, jsonData, 0644)

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
	if len(p.events) > 50000 {
		return errors.New("ДОСТИГНУТ ЛИМИТ ХРАНИЛИЩА. ПРОВЕРЬТЕ ПЕРЕДАЧУ")
	}
	p.append(events)
	return p.save()
}

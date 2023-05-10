package internalprocessor

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/windnow/tlanalyzer/internal/myfsm"
)

type InternalProcessor struct {
	cacheFile string
	events    []myfsm.Event
	mutex     sync.Mutex
	log       *logrus.Logger
	ctx       context.Context
}

func NewProcessor(ctx context.Context, log *logrus.Logger) (*InternalProcessor, error) {
	processor := &InternalProcessor{
		cacheFile: ".\\cache.out",
		ctx:       ctx,
		log:       log,
		events:    make([]myfsm.Event, 0),
	}

	processor.restore()
	go processor.startMonitoring()

	return processor, nil
}

func (p *InternalProcessor) startMonitoring() {

	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-p.ctx.Done():
			ticker.Stop()
			p.log.Info("Процесс мониторинга (внутренний) завершен по сигналу")
			return
		case <-ticker.C:
			p.log.Info("--internal scanner")
		}
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
	p.append(events)
	return p.save()
}

package redisprocessor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
	"github.com/windnow/tlanalyzer/internal/myfsm"
)

type RedisProcessor struct {
	rdb *redis.Client
	log *logrus.Logger
	ctx context.Context
}

func (p *RedisProcessor) Close() {
	p.log.Info("Закрываем соединение с redis")
	p.rdb.Close()
}

func NewProcessor(ctx context.Context, log *logrus.Logger) (*RedisProcessor, error) {
	type Config struct {
		Socket   string `json:"socket"`
		Password string `json:"password"`
		DB       int    `json:"db"`
	}
	file, err := os.Open("redis_config.json")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	config := Config{}
	err = decoder.Decode(&config)
	if err != nil {
		fmt.Println("error decoding config file:", err)
		return nil, err
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Socket,
		Password: config.Password,
		DB:       config.DB,
	})
	pong, err := rdb.Ping().Result()
	if err != nil {
		log.Errorf("Ошибка проверки связи с redis: %s", err.Error())
		return nil, err
	}
	log.Infof("Соединение с redis (%s, DB: %d) установлено: %s", config.Socket, config.DB, pong)

	proc := &RedisProcessor{
		log: log,
		rdb: rdb,
		ctx: ctx,
	}
	go proc.startMonitoring()
	return proc, nil
}

func (p *RedisProcessor) startMonitoring() {

	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-p.ctx.Done():
			ticker.Stop()
			p.log.Info("Процесс мониторинга redis завершен по сигналу")
			return
		case <-ticker.C:
			p.log.Info("--redis scanner")
		}
	}

}

func (p *RedisProcessor) SendEvents(events []myfsm.Event) error {
	if len(events) == 0 {
		return nil
	}

	eventsJSON, err := json.Marshal(events)
	if err != nil {
		return err
	}
	p.log.Info(string(eventsJSON))

	return errors.New("ЗАГЛУШКА")
}

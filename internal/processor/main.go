package processor

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
	"github.com/windnow/tlanalyzer/internal/myfsm"
)

type Processor interface {
	Close()
	SendEvents(events []myfsm.Event) error
}

type RedisProcessor struct {
	rdb *redis.Client
	log *logrus.Logger
}

func (p *RedisProcessor) Close() {
	p.rdb.Close()
}

func NewRedisProcessor(log *logrus.Logger) (*RedisProcessor, error) {
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

	return &RedisProcessor{
		log: log,
		rdb: rdb,
	}, nil
}

func (m *RedisProcessor) SendEvents(events []myfsm.Event) error {
	if len(events) == 0 {
		return nil
	}

	eventsJSON, err := json.Marshal(events)
	if err != nil {
		return err
	}
	m.log.Info(string(eventsJSON))

	return errors.New("ЗАГЛУШКА")
}

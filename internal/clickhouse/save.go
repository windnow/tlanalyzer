package clickhouse

import (
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/windnow/tlanalyzer/internal/myfsm"
)

func (ch *ClickHouse) Save(events []myfsm.Event) error {
	batch, err := ch.PrepareBatch(`INSERT INTO events`)
	if err != nil {
		return err
	}
	begin, err := time.Parse("2006-01-02", "1970-01-01")
	if err != nil {
		return err
	}
	for _, e := range events {
		event, ok := e.(*myfsm.BulkEvent)
		if !ok {
			return errors.New("НЕ УДАЛОСЬ ПРИВЕСТИ К ТИПУ БАЗОВОГО СОБЫТИЯ")
		}

		dur, err := strconv.Atoi(event.Duration)
		if err != nil {
			return errors.New("Не удалось преобразовать длительность события к числу")
		}
		idx := int32(event.Position)
		duration := int32(dur)
		context, _ := event.Fields["Context"]
		if begin.After(event.Time) {
			log.Printf("-------> Пропущено событие %s из за не корректной даты %s", event.Name, event.Time.Format("2006.01.02"))
			continue
		}
		err = batch.Append(
			event.Time,
			idx,
			event.Tag,
			event.Name,
			context,
			duration,
		)
		if err != nil {
			return err
		}
	}
	return batch.Send()
}

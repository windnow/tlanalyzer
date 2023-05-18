package clickhouse

import (
	"context"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/windnow/tlanalyzer/internal/myfsm"
)

func (ch *ClickHouse) Save(ctx context.Context, events []myfsm.Event) error {
	batch, err := ch.PrepareBatch(`INSERT INTO events`)
	if err != nil {
		return err
	}
	begin, err := time.Parse("2006-01-02", "1970-01-01")
	if err != nil {
		return err
	}
EventsProc:
	for _, e := range events {
		select {
		case <-ctx.Done():
			return errors.New("ЗАПРОС ПРЕРВАН")
		default:
			event, ok := e.(*myfsm.BulkEvent)
			if !ok {
				log.Println("НЕ УДАЛОСЬ ПРИВЕСТИ К ТИПУ БАЗОВОГО СОБЫТИЯ")
				continue EventsProc
			}

			dur, err := strconv.Atoi(event.Duration)
			if err != nil {
				return errors.New("НЕ УДАЛОСЬ ПРЕОБРАЗОВАТЬ ДЛИТЕЛЬНОСТЬ К ЧИСЛО")
			}
			idx := int32(event.Position)
			duration := int32(dur)
			context, _ := event.Fields["Context"]
			user, _ := event.Fields["Usr"]
			Sql, _ := event.Fields["Sql"]
			computerName, _ := event.Fields["t:computerName"]
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
				user, //+
				Sql,
				computerName,
				event.ProcessName,
				int32(event.ProcessPID),
			)
			if err != nil {
				return err
			}
		}
	}
	return batch.Send()
}

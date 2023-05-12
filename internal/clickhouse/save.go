package clickhouse

import (
	"errors"
	"fmt"

	"github.com/windnow/tlanalyzer/internal/myfsm"
)

func (ch *ClickHouse) Save(events []myfsm.Event) error {
	batch, err := ch.PrepareBatch(`INSERT INTO tlevents`)
	if err != nil {
		return err
	}
	for _, e := range events {
		event, ok := e.(*myfsm.BulkEvent)
		if !ok {
			return errors.New("НЕ УДАЛОСЬ ПРИВЕСТИ К ТИПУ БАЗОВОГО СОБЫТИЯ")
		}
		/*dur, err := strconv.Atoi(event.Duration)
		if err != nil {
			return errors.New("Не удалось преобразовать длительность события к числу")
		}*/
		idx := fmt.Sprintf("%d", event.Position)
		err = batch.Append(
			idx,
			event.Tag,
			event.Name,
			event.Duration,
		)
		if err != nil {
			return err
		}
	}
	return batch.Send()
}

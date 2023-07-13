package clickhouse

import (
	"context"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/windnow/tlanalyzer/internal/myfsm"
)

func getByKey(fields map[string]string, key string) (string, bool) {
	value, ok := fields[key]
	return value, ok
}

func getUint64(value string, ok bool) uint64 {

	if !ok {
		return 0
	}
	result, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0
	}
	return result

}
func getUint32(value string, ok bool) uint32 {

	if !ok {
		return 0
	}
	result, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0
	}

	return uint32(result)

}

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

			//--------------------------------------------------------------------------
			duration := getUint64(event.Duration, true)
			idx := int32(event.Position)
			context, _ := getByKey(event.Fields, "Context")
			user, _ := getByKey(event.Fields, "Usr")
			Sql, _ := getByKey(event.Fields, "Sql")
			computerName, _ := getByKey(event.Fields, "t:computerName")
			DataBase, _ := getByKey(event.Fields, "DataBase")
			dbPid := getUint32(getByKey(event.Fields, "dbpid"))
			SessionID := getUint32(getByKey(event.Fields, "SessionID"))
			MemoryPeak := getUint64(getByKey(event.Fields, "MemoryPeak"))
			CpuTime := getUint64(getByKey(event.Fields, "CpuTime"))
			//--------------------------------------------------------------------------

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
				SessionID,
				DataBase,
				dbPid,
				MemoryPeak,
				CpuTime,
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

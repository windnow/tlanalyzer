package clickhouse

import (
	"context"
	"fmt"
	"log"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type ClickHouse struct {
	ctx  context.Context
	conn driver.Conn
}

func (ch *ClickHouse) PrepareBatch(sql string) (driver.Batch, error) {
	return ch.conn.PrepareBatch(ch.ctx, sql)
}

func New(ctx context.Context) (*ClickHouse, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{"localhost:9000"},
		Auth: clickhouse.Auth{Database: "ytzh_db"},
		ClientInfo: clickhouse.ClientInfo{
			Products: []struct {
				Name    string
				Version string
			}{
				{Name: "1c-techlog-analyzer", Version: "0.1"},
			},
		},
		Debugf: func(format string, v ...any) {
			fmt.Printf(format, v)
		},
	})
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(ctx); err != nil {
		if excepation, ok := err.(*clickhouse.Exception); ok {
			fmt.Printf("Exception [%d] %s\n%s\n", excepation.Code, excepation.Message, excepation.StackTrace)
		}
		return nil, err
	}

	rows, err := conn.Query(ctx, "SELECT name,toString(uuid) as uuid_str FROM system.tables LIMIT 15")
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var name, uuid string
		if err := rows.Scan(&name, &uuid); err != nil {
			log.Printf("Error on read row data")
			continue
		}
		log.Printf("\tname: %s,\t uuid: %s\n", name, uuid)
	}

	cs := &ClickHouse{conn: conn, ctx: ctx}
	return cs, nil
}

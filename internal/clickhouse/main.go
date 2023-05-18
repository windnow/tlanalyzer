package clickhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type Config struct {
	Addr []string        `json:"addr"`
	Auth clickhouse.Auth `json:"auth"`
}

type ClickHouse struct {
	ctx    context.Context
	conn   driver.Conn
	tables map[string]*table
}

func getTable(tname, createQuery string) *table {

	t := &table{fields: make(map[string]field)}
	t.parseFields(createQuery)
	t.setName(tname)
	return t
}

func (ch *ClickHouse) parseTables(tables map[string]string) {

	for tname, createQuery := range tables {
		ch.tables[tname] = getTable(tname, createQuery)
	}
}

func (ch *ClickHouse) PrepareBatch(sql string) (driver.Batch, error) {
	return ch.conn.PrepareBatch(ch.ctx, sql)
}

func getConfig() Config {
	config := Config{}
	data, err := os.ReadFile("config/clickhouse.json")
	if err != nil {
		log.Println("Не удалось прочитать файл конфигурации ClickHouse (`config/clickhouse.json`). Параметры установлены по умолчанию")
	} else {
		err = json.Unmarshal(data, &config)
		if err != nil {
			log.Println("Не удалось разобрать файл конфигурации ClickHouse. Параметры установлены по умолчанию")
		}
	}
	if len(config.Addr) == 0 {
		config.Addr = append(config.Addr, "localhost:9000")
	}
	if config.Auth.Database == "" {
		config.Auth.Database = "default"
	}

	return config

}

func New(ctx context.Context) (*ClickHouse, error) {
	config := getConfig()
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: config.Addr,
		Auth: config.Auth,
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

	rows, err := conn.Query(ctx, fmt.Sprintf("SELECT name, toString(uuid) as uuid_str, create_table_query FROM system.tables where database = '%s'", config.Auth.Database))
	if err != nil {
		return nil, err
	}

	info := fmt.Sprintf("Список таблиц базы `%s`\n", config.Auth.Database)

	tables := make(map[string]string, 0)
	for rows.Next() {
		var name, uuid, query string
		if err := rows.Scan(&name, &uuid, &query); err != nil {
			log.Printf("Error on read row data")
			continue
		}
		info = fmt.Sprintf("%s\tname: %s,\t uuid: %s\n\t%s\n", info, name, uuid, query)
		tables[name] = query
	}
	log.Println(info)

	cs := &ClickHouse{conn: conn, ctx: ctx, tables: make(map[string]*table)}
	cs.parseTables(tables)
	return cs, nil
}

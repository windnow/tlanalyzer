package clickhouse

import (
	"fmt"
	"strings"
	"sync"
)

type field struct {
	ftype string
}

type table struct {
	fields  map[string]field
	sysname string
	mutex   sync.Mutex
}

func (t *table) setName(name string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.sysname = name
}

func (t *table) getName() string {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.sysname
}

func (t *table) parseFields(create_table_query string) error {

	t.fields = make(map[string]field)
	ctq := strings.ReplaceAll(create_table_query, "`", "")
	ctq = ctq[strings.IndexRune(ctq, '(')+1:]
	ctq = ctq[:strings.IndexRune(ctq, ')')]

	fields := strings.Split(ctq, ", ")
	for _, discr := range fields {
		tmp := strings.Split(discr, " ")
		if len(tmp) != 2 {
			return fmt.Errorf("НЕ ВЕРНО ЗАДАНЫ ПОЛЯ (%s)", create_table_query)
		}
		t.fields[strings.TrimSpace(tmp[0])] = field{ftype: strings.TrimSpace(tmp[1])}
	}

	return nil

}

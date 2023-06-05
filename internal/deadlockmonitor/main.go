package deadlockmonitor

import (
	"context"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/sirupsen/logrus"
	"github.com/windnow/tlanalyzer/internal/myfsm"
)

type Event interface {
	getEvent() myfsm.Event
}

type deadlockmonitor struct {
	config  *config
	offsets []fileOffset

	workDir string
	log     *logrus.Logger
	ctx     context.Context
}

type config struct {
	LogsDir    string `json:"logs-dir"`
	ConnString string `json:"connection-string"`
}

type fileOffset struct {
	fileName           string
	lastEventTimestamp time.Time
}

func New(logsDir, ctx context.Context, workDir string, log *logrus.Logger) (*deadlockmonitor, error) {

	p := &deadlockmonitor{
		workDir: workDir,
		ctx:     ctx,
		log:     log,
	}

	if err := p.loadConfig(); err != nil {
		return nil, err
	}
	p.loadOffsets()

	return p, nil

}

func (p *deadlockmonitor) loadConfig() error {

	conf := &config{}

	data, err := os.ReadFile(fmt.Sprintf("%s/config/deadlockmonitor_config.json", p.workDir))
	if err != nil {
		p.log.Warn("Не удалось прочитать файл конфигурации мониторинга взаимоблокировок.")
	} else {
		err = json.Unmarshal(data, conf)
		if err != nil {
			p.log.Warn("Не удалось разобрать файл конфигурации мониторинга взаимоблокировок.")
		}
	}

	p.config = conf
	return nil
}

func (p *deadlockmonitor) loadOffsets() {
	data, err := os.ReadFile(fmt.Sprintf("%s/config/deadlockmonitor_offsets.json", p.workDir))
	if err != nil {
		p.offsets = make([]fileOffset, 0)
		return
	}
	if err = json.Unmarshal(data, &p.offsets); err != nil {
		p.offsets = make([]fileOffset, 0)
		return
	}

}

func (p *deadlockmonitor) read(fileName string, timestamp time.Time) ([]myfsm.Event, error) {
	filename := "D:\\shared\\temp\\DeadlockAnalyze_0_133304108659730000.xel"

	db, err := sql.Open("mssql", p.config.ConnString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	query := `SELECT
		timestamp_utc,
		CONVERT(XML, event_data)
	FROM
		sys.fn_xe_file_target_read_file(@filename, null, null, null) 
	WHERE 
		object_name = 'xml_deadlock_report' 
		AND timestamp_utc > @timestamp
		AND event_data like '%victimProcess id=%'
	`

	rows, err := db.QueryContext(ctx, query, sql.Named("filename", filename), sql.Named("timestamp", timestamp))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]myfsm.Event, 0)

	for rows.Next() {
		event, err := p.parseRow(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event.getEvent())
	}

	return events, nil
}

func (p *deadlockmonitor) parseRow(rows *sql.Rows) (Event, error) {

	var timestamp time.Time
	var eventData string

	if err := rows.Scan(&timestamp, &eventData); err != nil {
		return nil, err
	}
	var e event

	if err := xml.Unmarshal([]byte(eventData), &e); err != nil {
		return nil, err
	}

	return &e, nil
}

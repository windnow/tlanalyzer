package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"log"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
)

func main() {
	host := "localhost"
	filename := "D:\\shared\\temp\\DeadlockAnalyze_0_133304108659730000.xel"
	connString := fmt.Sprintf("server=%s;integrated security=true;", host)

	db, err := sql.Open("mssql", connString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	query := fmt.Sprintf(`SELECT
		timestamp_utc,
		CONVERT(XML, event_data)
	FROM
		sys.fn_xe_file_target_read_file('%s', null, null, null) 
	WHERE 
		object_name = 'xml_deadlock_report' 
		AND event_data like '%%victimProcess id=%%'
	`, filename)

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		parseRow(rows)
	}
}

type event struct {
	XMLName   xml.Name     `xml:"event"`
	Name      string       `xml:"name,attr"`
	Timestamp time.Time    `xml:"timestamp,attr"`
	Victims   []victim     `xml:"data>value>deadlock>victim-list>victimProcess"`
	Processes []process    `xml:"data>value>deadlock>process-list>process"`
	Resources []objectlock `xml:"data>value>deadlock>resource-list>objectlock"`
}

type victim struct {
	Id string `xml:"id,attr"`
}

type process struct {
	Id             string   `xml:"id,attr"`
	Waittime       int      `xml:"waittime,attr"`
	OwnerId        string   `xml:"ownerId,attr"`
	LockMode       string   `xml:"lockMode,attr"`
	Schedulerid    int      `xml:"schedulerid,attr"`
	Hostname       string   `xml:"hostname,attr"`
	Hostpid        int      `xml:"hostpid,attr"`
	SPID           int      `xml:"spid,attr"`
	Loginname      string   `xml:"loginname,attr"`
	Isolationlevel string   `xml:"isolationlevel,attr"`
	Inputbuf       inputbuf `xml:"inputbuf"`
}

type objectlock struct {
	Objectname string        `xml:"objectname,attr"`
	Mode       string        `xml:"mode,attr"`
	Owners     []lockprocess `xml:"owner-list>owner"`
	Waiters    []lockprocess `xml:"waiter-list>waiter"`
}

type lockprocess struct {
	Id          string `xml:"id,attr"`
	Mode        string `xml:"mode,attr"`
	RequestType string `xml:"requestType,attr,omitempty"`
}

type inputbuf struct {
	Value string `xml:",chardata"`
}

func parseRow(rows *sql.Rows) (*event, error) {

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

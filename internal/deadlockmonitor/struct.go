package deadlockmonitor

import (
	"encoding/xml"
	"time"

	"github.com/windnow/tlanalyzer/internal/myfsm"
)

type event struct {
	XMLName   xml.Name     `xml:"event"`
	Name      string       `xml:"name,attr"`
	Timestamp time.Time    `xml:"timestamp,attr"`
	Victims   []victim     `xml:"data>value>deadlock>victim-list>victimProcess"`
	Processes []process    `xml:"data>value>deadlock>process-list>process"`
	Resources []objectlock `xml:"data>value>deadlock>resource-list>objectlock"`
}

func (e *event) getEvent() myfsm.Event {
	return nil
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

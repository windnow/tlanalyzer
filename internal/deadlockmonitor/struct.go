package deadlockmonitor

import (
	"encoding/xml"
	"time"
)

type event struct {
	XMLName   xml.Name  `xml:"event"`
	Name      string    `xml:"name,attr"`
	Timestamp time.Time `xml:"timestamp,attr"`

	Victims     []processRef `xml:"data>value>deadlock>victim-list>victimProcess"`
	Processes   []process    `xml:"data>value>deadlock>process-list>process"`
	Objectlocks []objectlock `xml:"data>value>deadlock>resource-list>objectlock"`
	Pagelocks   []objectlock `xml:"data>value>deadlock>resource-list>pagelock"`
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
	Objectname string       `xml:"objectname,attr"`
	Mode       string       `xml:"mode,attr"`
	Owners     []processRef `xml:"owner-list>owner"`
	Waiters    []processRef `xml:"waiter-list>waiter"`
}

type processRef struct {
	Id          string `xml:"id,attr"`
	Mode        string `xml:"mode,attr,omitempty"`
	RequestType string `xml:"requestType,attr,omitempty"`
}

type inputbuf struct {
	Value string `xml:",chardata"`
}

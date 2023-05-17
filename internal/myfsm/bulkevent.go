package myfsm

import (
	"strconv"
	"strings"
	"time"

	"github.com/windnow/tlanalyzer/internal/common"
)

type BulkEvent struct {
	time        string
	Position    int               `json:"idx"`
	Tag         string            `json:"tag"`
	Time        time.Time         `json:"time"`
	Duration    string            `json:"duration"`
	Name        string            `json:"name"`
	Level       string            `json:"level"`
	Fields      map[string]string `json:"fields"`
	ProcessName string            `json:"procName"`
	ProcessPID  int               `json:"procPID"`
}

func (e BulkEvent) GetField(fieldName string) *string {
	var result *string
	var buf string

	switch fieldName {
	case "time":
		result = &e.time
	case "duration":
		result = &e.Duration
	case "name":
		result = &e.Name
	case "level":
		result = &e.Level
	default:
		val, ok := e.Fields[fieldName]
		if ok {
			result = &val
		}
	}

	if result != nil {
		buf = *result
		result = &buf
	}
	return result
}

func (e *BulkEvent) SetField(fieldName, value string) {

	switch fieldName {
	case "time":
		e.time = value
	case "duration":
		e.Duration = value
	case "name":
		e.Name = value
	case "level":
		e.Level = value
	default:
		e.Fields[fieldName] = value
	}
}

func (e *BulkEvent) ParseTime(loc *time.Location) error {
	t, err := time.ParseInLocation("20060102 15:04:05.000000", e.time, loc)
	if err != nil {
		return err
	}
	e.Time = t
	return nil
}

func (e *BulkEvent) SetIndex(i int) {
	e.Position = i
}

func (e *BulkEvent) SetTag(tag string) {
	e.Tag = tag
}

func (i *BulkEvent) ParsePath(path string) {

	parent, _ := common.GetParentDirectoryName(path)
	parts := strings.Split(parent, "_")

	if len(parts) == 2 {
		pid, err := strconv.Atoi(parts[1])
		if err != nil {
			return
		}
		i.ProcessName = parts[0]
		i.ProcessPID = pid
	}
}

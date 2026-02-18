package myfsm

import (
	"regexp"
	"strings"
	"time"
)

type Event interface {
	GetField(string) *string
	SetField(string, string)
	ParseTime(*time.Location) error
	SetIndex(int)
	SetTag(string)
	ParsePath(string)
}

type Process func()

type myFSM struct {
	reNewEvent   *regexp.Regexp
	c            rune
	prev_c       rune
	quoter       rune
	fileName     string
	currentField string
	events       []Event
	valueBuilder strings.Builder
	bufBuilder   strings.Builder
	Event        Process
	tail         string
	full         bool
}

func (f *myFSM) Tail() string {
	return f.tail
}

func (f *myFSM) Full() bool {
	return f.full
}

func (f *myFSM) GetEvents() []Event {
	return f.events
}
func NewFSM(fileName string) *myFSM {
	r := regexp.MustCompile(`^\d\d:\d\d\.\d{6}`)
	return &myFSM{
		fileName:   fileName,
		reNewEvent: r,
	}
}

func (fsm *myFSM) NewEvent() {
	if fsm.valueBuilder.Len() == 0 {
		if fsm.prev_c > 0 {
			fsm.valueBuilder.WriteRune(fsm.prev_c)
			fsm.prev_c = 0
		}
		if len(fsm.events) >= 5000 {
			fsm.full = true
			return
		}
		fsm.events = append(fsm.events, &BulkEvent{
			Fields: make(map[string]string),
		})
	}
	if (fsm.c >= '0' && fsm.c <= '9') || (fsm.c == '.' && fsm.prev_c >= '0' && fsm.prev_c <= '9') || (fsm.c == ':' && fsm.prev_c >= '0' && fsm.prev_c <= '9') {
		fsm.valueBuilder.WriteRune(fsm.c)
		fsm.prev_c = fsm.c
	} else {
		fsm.events[len(fsm.events)-1].SetField("time", fsm.fileName[:8]+" "+fsm.fileName[8:]+":"+fsm.valueBuilder.String())
		fsm.prev_c = 0
		fsm.valueBuilder.Reset()
		fsm.Event = fsm.DurationEvent
	}
}

func (fsm *myFSM) DurationEvent() {
	if fsm.c >= '0' && fsm.c <= '9' {
		fsm.valueBuilder.WriteRune(fsm.c)
	} else {
		fsm.events[len(fsm.events)-1].SetField("duration", fsm.valueBuilder.String())
		fsm.valueBuilder.Reset()
		fsm.Event = fsm.NameEvent
	}
}
func (fsm *myFSM) NameEvent() {
	if (fsm.c >= 'a' && fsm.c <= 'z') || (fsm.c >= 'A' && fsm.c <= 'Z') {
		fsm.valueBuilder.WriteRune(fsm.c)
	} else {
		fsm.events[len(fsm.events)-1].SetField("name", fsm.valueBuilder.String())
		fsm.valueBuilder.Reset()
		fsm.Event = fsm.LevelEvent
	}
}
func (fsm *myFSM) ProcessLine(line string) {
	line = strings.TrimSpace(line)
	if fsm.reNewEvent.MatchString(line) {
		if fsm.Event != nil {
			fsm.Event = fsm.FinalizeEvent
		} else {
			fsm.Event = fsm.NewEvent
		}
	} else {
		line = "\n" + line
	}

	for _, c := range line {
		fsm.Update(c)
		if fsm.full {
			fsm.tail = line
			break
		}
	}

}

func (fsm *myFSM) LevelEvent() {
	if fsm.c >= '0' && fsm.c <= '9' {
		fsm.valueBuilder.WriteRune(fsm.c)
	} else {
		fsm.events[len(fsm.events)-1].SetField("level", fsm.valueBuilder.String())
		fsm.valueBuilder.Reset()
		fsm.Event = fsm.FieldEvent
	}
}

func (fsm *myFSM) FieldEvent() {
	if fsm.valueBuilder.Len() == 0 && fsm.c == ',' {
		return
	}
	if fsm.c != '=' {
		fsm.valueBuilder.WriteRune(fsm.c)
	} else {
		fsm.currentField = fsm.valueBuilder.String()
		fsm.valueBuilder.Reset()
		fsm.prev_c = 0
		fsm.Event = fsm.ValueEvent
	}
}

func (fsm *myFSM) ValueEvent() {
	if (fsm.c == '\'' || fsm.c == '"') && fsm.prev_c == 0 {

		fsm.quoter = fsm.c

		fsm.valueBuilder.Reset()
		fsm.Event = fsm.QuotedValueEvent

	} else if fsm.c == ',' {
		fsm.events[len(fsm.events)-1].SetField(fsm.currentField, strings.TrimSpace(fsm.valueBuilder.String()))

		fsm.prev_c = 0
		fsm.currentField = ""
		fsm.valueBuilder.Reset()
		fsm.Event = fsm.FieldEvent
	} else {
		fsm.valueBuilder.WriteRune(fsm.c)
		fsm.prev_c = fsm.c
	}
}
func (fsm *myFSM) endQuotedValueEvent() {
	if fsm.c == ',' {
		fsm.events[len(fsm.events)-1].SetField(fsm.currentField, strings.TrimSpace(fsm.valueBuilder.String()))
		fsm.currentField = ""
		fsm.valueBuilder.Reset()
		fsm.bufBuilder.Reset()
		fsm.Event = fsm.FieldEvent
	} else {
		fsm.valueBuilder.WriteString(fsm.bufBuilder.String())
		fsm.valueBuilder.WriteRune(fsm.c)
		fsm.bufBuilder.Reset()
		fsm.Event = fsm.QuotedValueEvent
	}
	fsm.bufBuilder.Reset()
}

func (fsm *myFSM) QuotedValueEvent() {
	if fsm.c == fsm.quoter {
		fsm.bufBuilder.WriteRune(fsm.c)
		fsm.Event = fsm.endQuotedValueEvent
	} else {
		fsm.valueBuilder.WriteRune(fsm.c)
		fsm.prev_c = fsm.c
	}

}

func (fsm *myFSM) Update(c rune) {
	fsm.c = c
	if fsm.Event != nil {
		fsm.Event()
	}
}

func (fsm *myFSM) FinalizeEvent() {
	if len(fsm.events) > 0 && len(fsm.currentField) > 0 && fsm.valueBuilder.Len() > 0 {
		fsm.events[len(fsm.events)-1].SetField(fsm.currentField, fsm.valueBuilder.String())
		fsm.valueBuilder.Reset()
	}
	fsm.prev_c = fsm.c
	fsm.Event = fsm.NewEvent
}

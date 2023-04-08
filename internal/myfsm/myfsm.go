package myfsm

import "strings"

type Event struct {
	time     string
	duration string
	name     string
	level    string
	Fields   map[string]string
}

type Process func()

type myFSM struct {
	c            rune
	prev_c       rune
	quoter       rune
	fileName     string
	currentField string
	events       []*Event
	value        string
	Event        Process
}

func (fsm *myFSM) NewEvent() {
	if len(fsm.value) == 0 {
		if fsm.prev_c > 0 {
			fsm.value = string(fsm.prev_c)
			fsm.prev_c = 0
		}
		fsm.events = append(fsm.events, &Event{
			Fields: make(map[string]string),
		})
	}
	if (fsm.c >= '0' && fsm.c <= '9') || (fsm.c == '.' && fsm.prev_c >= '0' && fsm.prev_c <= '9') || (fsm.c == ':' && fsm.prev_c >= '0' && fsm.prev_c <= '9') {
		fsm.value += string(fsm.c)
		fsm.prev_c = fsm.c
	} else {
		fsm.events[len(fsm.events)-1].time = fsm.fileName[:8] + " " + fsm.fileName[8:] + ":" + fsm.value
		fsm.prev_c = 0
		fsm.value = ""
		fsm.Event = fsm.DurationEvent
	}
}

func (fsm *myFSM) DurationEvent() {
	if fsm.c >= '0' && fsm.c <= '9' {
		fsm.value += string(fsm.c)
	} else {
		fsm.events[len(fsm.events)-1].duration = fsm.value
		fsm.value = ""
		fsm.Event = fsm.NameEvent
	}
}
func (fsm *myFSM) NameEvent() {
	if (fsm.c >= 'a' && fsm.c <= 'z') || (fsm.c >= 'A' && fsm.c <= 'Z') {
		fsm.value += string(fsm.c)
	} else {
		fsm.events[len(fsm.events)-1].name = fsm.value
		fsm.value = ""
		fsm.Event = fsm.LevelEvent
	}
}

func (fsm *myFSM) LevelEvent() {
	if fsm.c >= '0' && fsm.c <= '9' {
		fsm.value += string(fsm.c)
	} else {
		fsm.events[len(fsm.events)-1].level = fsm.value
		fsm.value = ""
		fsm.Event = fsm.FieldEvent
	}
}

func (fsm *myFSM) FieldEvent() {
	if len(fsm.value) == 0 && fsm.c == ',' {
		return
	}
	if fsm.c != '=' {
		fsm.value += string(fsm.c)
	} else {
		fsm.currentField = fsm.value
		fsm.value = ""
		fsm.prev_c = 0
		fsm.Event = fsm.ValueEvent
	}
}

func (fsm myFSM) GetOne(num int) string {
	return fsm.events[num].Fields["Sql"] + "\n\n" + fsm.events[num].Fields["Context"]
}

func (fsm *myFSM) ValueEvent() {
	if (fsm.c == '\'' || fsm.c == '"') && fsm.prev_c == 0 {

		fsm.quoter = fsm.c

		fsm.value = ""
		fsm.Event = fsm.QuotedValueEvent

	} else if fsm.c == ',' {
		fsm.events[len(fsm.events)-1].Fields[fsm.currentField] = strings.TrimSpace(fsm.value)
		fsm.value = ""
		fsm.Event = fsm.FieldEvent
	} else {
		fsm.value += string(fsm.c)
		fsm.prev_c = fsm.c
	}
}

func (fsm *myFSM) QuotedValueEvent() {
	if fsm.c == fsm.quoter && fsm.prev_c != '\\' {
		fsm.events[len(fsm.events)-1].Fields[fsm.currentField] = strings.TrimSpace(fsm.value)

		fsm.value = ""
		fsm.Event = fsm.FieldEvent
	} else {
		fsm.value += string(fsm.c)
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
	if len(fsm.events) > 0 && len(fsm.currentField) > 0 && len(fsm.value) > 0 {
		fsm.events[len(fsm.events)-1].Fields[fsm.currentField] = fsm.value
		fsm.value = ""
	}
	fsm.prev_c = fsm.c
	fsm.Event = fsm.NewEvent
}

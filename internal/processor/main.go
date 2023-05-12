package processor

import (
	"github.com/windnow/tlanalyzer/internal/myfsm"
)

type Processor interface {
	Close()
	SendEvents(events []myfsm.Event) error
}

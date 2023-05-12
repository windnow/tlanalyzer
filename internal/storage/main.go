package storage

import "github.com/windnow/tlanalyzer/internal/myfsm"

type Storage interface {
	Save([]myfsm.Event) error
}

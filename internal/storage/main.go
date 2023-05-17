package storage

import (
	"context"

	"github.com/windnow/tlanalyzer/internal/myfsm"
)

type Storage interface {
	Save(context.Context, []myfsm.Event) error
}

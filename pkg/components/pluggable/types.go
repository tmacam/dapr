package pluggable

import (
	s "github.com/dapr/components-contrib/state"
	"github.com/dapr/dapr/pkg/components/state"
)

type DaprComponent interface {
	Name() string
	Version() string
	Close() error
}

type StateStoreComponent interface {
	DaprComponent
	s.Store
	StateStore() state.State
}

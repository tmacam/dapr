package pluggable

import (
	b "github.com/dapr/components-contrib/bindings"
	s "github.com/dapr/components-contrib/state"
	"github.com/dapr/dapr/pkg/components/bindings"
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

type InputBindingComponent interface {
	DaprComponent
	b.InputBinding
	InputBinding() bindings.InputBinding
}

type OutputBindingComponent interface {
	DaprComponent
	b.OutputBinding
	OutputBinding() bindings.OutputBinding
}

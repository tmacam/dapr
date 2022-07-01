package pluggable

import (
	b "github.com/dapr/components-contrib/bindings"
	p "github.com/dapr/components-contrib/pubsub"
	s "github.com/dapr/components-contrib/state"
	"github.com/dapr/dapr/pkg/components/bindings"
	"github.com/dapr/dapr/pkg/components/pubsub"
	"github.com/dapr/dapr/pkg/components/state"
	"github.com/dapr/kit/logger"
)

var (
	log = logger.NewLogger("pluggable-components")
)

type GRPCComponent interface {
	GRPCClient()
}

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

type PubSubComponent interface {
	DaprComponent
	p.PubSub
	PubSub() pubsub.PubSub
}

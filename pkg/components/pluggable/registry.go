package pluggable

import (
	"github.com/dapr/dapr/pkg/components/state"
	"github.com/dapr/dapr/pkg/runtime"
)

type ComponentRegistry struct {
	stateStores []StateStoreComponent
}

func NewComponentRegistry() *ComponentRegistry {
	return &ComponentRegistry{
		stateStores: make([]StateStoreComponent, 0),
	}
}

func (r *ComponentRegistry) AddStateStore(ss StateStoreComponent) {
	r.stateStores = append(r.stateStores, ss)
}

func (r *ComponentRegistry) StateStores() []state.State {
	stores := make([]state.State, len(r.stateStores))
	for _, s := range r.stateStores {
		stores = append(stores, s.StateStore())
	}
	return stores
}

func (r *ComponentRegistry) GenerateRuntimeOptions() []runtime.Option {
	return []runtime.Option{
		runtime.WithStates(
			r.StateStores()...,
		),
	}
}

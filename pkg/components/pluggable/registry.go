package pluggable

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	components_v1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	"github.com/ghodss/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/dapr/dapr/pkg/components/bindings"
	"github.com/dapr/dapr/pkg/components/pubsub"
	"github.com/dapr/dapr/pkg/components/state"
)

type ComponentRegistry struct {
	stateStores    []StateStoreComponent
	pubsubs        []PubSubComponent
	inputBindings  []InputBindingComponent
	outputBindings []OutputBindingComponent
}

func NewComponentRegistry() *ComponentRegistry {
	return &ComponentRegistry{
		stateStores:    make([]StateStoreComponent, 0),
		pubsubs:        make([]PubSubComponent, 0),
		inputBindings:  make([]InputBindingComponent, 0),
		outputBindings: make([]OutputBindingComponent, 0),
	}
}

func (r *ComponentRegistry) AddInputBinding(ib InputBindingComponent) {
	r.inputBindings = append(r.inputBindings, ib)
}

func (r *ComponentRegistry) InputBindings() []bindings.InputBinding {
	inputs := make([]bindings.InputBinding, len(r.inputBindings))
	for _, i := range r.inputBindings {
		inputs = append(inputs, i.InputBinding())
	}
	return inputs
}

func (r *ComponentRegistry) AddStateStore(ss StateStoreComponent) {
	r.stateStores = append(r.stateStores, ss)
}

func (r *ComponentRegistry) AddPubSub(ps PubSubComponent) {
	r.pubsubs = append(r.pubsubs, ps)
}

func (r *ComponentRegistry) StateStores() []state.State {
	stores := make([]state.State, len(r.stateStores))
	for _, s := range r.stateStores {
		stores = append(stores, s.StateStore())
	}
	return stores
}

func (r *ComponentRegistry) PubSubs() []pubsub.PubSub {
	pubsubs := make([]pubsub.PubSub, len(r.pubsubs))
	for _, p := range r.pubsubs {
		pubsubs = append(pubsubs, p.PubSub())
	}
	return pubsubs
}

func (r *ComponentRegistry) LoadComponentsFromPath(path string) error {
	log.Debugf("Loading pluggable components from path %s", path)
	if list, err := r.loadPluggableComponentsFromPath(path); err != nil {
		return fmt.Errorf("unable to load pluggable components from path %s: %v", path, err)
	} else {
		log.Debugf("Found %d pluggable components...", len(list))
		for _, pc := range list {
			log.Debugf(" * %s", pc.Name)
			r.RegisterPluggableComponent(pc)
		}
		return nil
	}
}

func (r *ComponentRegistry) RegisterPluggableComponent(pc components_v1alpha1.PluggableComponent) error {
	var socketPath string

	if pc.Spec.SocketPath != "" {
		socketPath = pc.Spec.SocketPath
	} else if pc.Spec.Container.Image != "" {
		log.Debugf("Starting container for pluggable component %s -- %s:%s...", pc.Name, pc.Spec.Container.Image, pc.Spec.Container.Version)
		cf := ContainerFactory{
			Image:          pc.Spec.Container.Image,
			Version:        pc.Spec.Container.Version,
			HostSocketRoot: "/home/johnewart/Temp/sockets",
		}
		if container, err := cf.StartContainer(context.TODO()); err != nil {
			log.Warnf("Unable to create container: %v", err)
			return err
		} else {
			// TODO no thanks...
			log.Debugf("Waiting 5 seconds for the container to come up...")
			time.Sleep(5 * time.Second)
			socketPath = container.HostSocketPath
		}
	}

	if socketPath != "" {
		log.Debugf("Using UNIX socket located at %s for pluggable component %s/%s", pc.Spec.SocketPath, pc.Name, pc.Spec.Version)

		switch pc.Spec.Type {
		case "state":
			if ss, err := NewGRPCStateStore(pc.Name, pc.Spec.Version, socketPath); err != nil {
				log.Warnf("Unable to create store GRPC component '%s': %v", err)
				return err
			} else {
				log.Debugf("Registered state store %s/%s", pc.Name, pc.Spec.Version)
				r.AddStateStore(ss)
			}
		case "pubsub":
			if ps, err := NewGRPCPubSub(pc.Name, pc.Spec.Version, socketPath); err != nil {
				log.Warnf("Unable to create GRPC pubsub component '%s': %v", err)
				return err
			} else {
				log.Debugf("Registering pubsub %s/%s", pc.Name, pc.Spec.Version)
				r.AddPubSub(ps)
			}
		case "inputbinding":
			if ib, err := NewGRPCInputBinding(pc.Name, pc.Spec.Version, socketPath); err != nil {
				log.Warnf("Unable to create GRPC input binding component '%s': %v", err)
				return err
			} else {
				log.Debugf("Registering input binding %s/%s", pc.Name, pc.Spec.Version)
				r.AddInputBinding(ib)
			}

		}
	}

	return nil
}

func (r *ComponentRegistry) loadPluggableComponentsFromPath(path string) ([]components_v1alpha1.PluggableComponent, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	list := []components_v1alpha1.PluggableComponent{}

	for _, file := range files {
		if !file.IsDir() && r.isYaml(file.Name()) {
			fpath := filepath.Join(path, file.Name())
			components := r.loadPluggableComponentsFromFile(fpath)
			if len(components) > 0 {
				list = append(list, components...)
			}
		}
	}

	return list, nil
}

const (
	yamlSeparator          = "\n---"
	pluggableComponentKind = "PluggableComponent"
)

func (r *ComponentRegistry) isYaml(fileName string) bool {
	extension := strings.ToLower(filepath.Ext(fileName))
	if extension == ".yaml" || extension == ".yml" {
		return true
	}
	return false
}

func (r *ComponentRegistry) loadPluggableComponentsFromFile(path string) []components_v1alpha1.PluggableComponent {
	var errors []error
	log.Debugf("Looking for pluggable components from file %s", path)
	components := []components_v1alpha1.PluggableComponent{}

	b, err := os.ReadFile(path)
	if err != nil {
		log.Warnf("daprd load components error when reading file %s : %s", path, err)
		return components
	}
	components, errors = r.decodePluggableComponentsInYaml(b)
	for _, err := range errors {
		log.Warnf("daprd load components error when parsing components yaml resource in %s : %s", path, err)
	}
	return components
}

func (cr *ComponentRegistry) decodePluggableComponentsInYaml(b []byte) ([]components_v1alpha1.PluggableComponent, []error) {
	list := []components_v1alpha1.PluggableComponent{}
	errors := []error{}
	scanner := bufio.NewScanner(bytes.NewReader(b))
	scanner.Split(cr.splitYamlDoc)

	type typeInfo struct {
		metav1.TypeMeta `json:",inline"`
	}

	for {
		if !scanner.Scan() {
			err := scanner.Err()
			if err != nil {
				errors = append(errors, err)

				continue
			}

			break
		}

		scannerBytes := scanner.Bytes()
		var ti typeInfo
		if err := yaml.Unmarshal(scannerBytes, &ti); err != nil {
			errors = append(errors, err)

			continue
		}

		if ti.Kind != pluggableComponentKind {
			log.Debugf("%s != %s, skipping", ti.Kind, pluggableComponentKind)
			continue
		}

		var comp components_v1alpha1.PluggableComponent
		if err := yaml.Unmarshal(scannerBytes, &comp); err != nil {
			errors = append(errors, err)

			continue
		}

		list = append(list, comp)
	}

	return list, errors
}

// splitYamlDoc - splits the yaml docs.
func (cr *ComponentRegistry) splitYamlDoc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	sep := len([]byte(yamlSeparator))
	if i := bytes.Index(data, []byte(yamlSeparator)); i >= 0 {
		i += sep
		after := data[i:]

		if len(after) == 0 {
			if atEOF {
				return len(data), data[:len(data)-sep], nil
			}
			return 0, nil, nil
		}
		if j := bytes.IndexByte(after, '\n'); j >= 0 {
			return i + j + 1, data[0 : i-sep], nil
		}
		return 0, nil, nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

package pluggable

import (
	"context"
	"fmt"
	"io"

	b "github.com/dapr/components-contrib/bindings"
	"github.com/dapr/dapr/pkg/components/bindings"
	proto "github.com/dapr/dapr/pkg/proto/components/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type GRPCInputBinding struct {
	GRPCComponent
	InputBindingComponent
	name    string
	version string
	client  proto.InputBindingClient
	context context.Context
}

func NewGRPCInputBinding(name string, version string, socketPath string) (*GRPCInputBinding, error) {
	socket := fmt.Sprintf("unix://%s", socketPath)
	if c, err := grpc.Dial(socket, grpc.WithInsecure()); err != nil {
		return nil, fmt.Errorf("unable to open GRPC connection using socket '%s': %v", socket, err)
	} else {
		return &GRPCInputBinding{
			name:    name,
			version: version,
			client:  proto.NewInputBindingClient(c),
			context: context.TODO(),
		}, nil
	}
}

func (ib *GRPCInputBinding) Name() string {
	return ib.name
}

func (ib *GRPCInputBinding) Version() string {
	return ib.version
}

func (ib *GRPCInputBinding) InputBinding() bindings.InputBinding {
	return bindings.InputBinding{
		Name: ib.Name(),
		FactoryMethod: func() b.InputBinding {
			return ib
		},
	}
}

func (ib *GRPCInputBinding) Close() error {
	return nil
}

func (ib *GRPCInputBinding) Init(metadata b.Metadata) error {
	log.Infof("Input binding %s initializing...", metadata.Name)

	protoMetadata := &proto.MetadataRequest{
		Properties: map[string]string{},
	}
	for k, v := range metadata.Properties {
		protoMetadata.Properties[k] = v
	}

	_, err := ib.client.Init(context.TODO(), protoMetadata)
	return err
}

func (ib *GRPCInputBinding) Ping() error {
	_, err := ib.client.Ping(ib.context, &emptypb.Empty{})
	return err
}

func (ib *GRPCInputBinding) Read(handler func(context.Context, *b.ReadResponse) ([]byte, error)) error {
	stream, err := ib.client.Read(context.Background(), &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("unable to subscribe: %v", err)
	}

	streamCtx := stream.Context()
	done := make(chan bool)

	// Read messages from the topic
	go func() error {
		for {
			event, err := stream.Recv()
			if err == io.EOF {
				close(done)
				return nil
			}
			if err != nil {
				return fmt.Errorf("failed to receive message: %v", err)
			}
			log.Debugf("Received message from input handler %s", ib.Name())
			m := b.ReadResponse{
				Data:        event.Data,
				Metadata:    event.Metadata,
				ContentType: &event.Contenttype,
			}
			handler(streamCtx, &m)
		}
	}()

	return nil
}

package main

import (
	"context"
	"fmt"

	b "github.com/dapr/components-contrib/bindings"
	"github.com/dapr/dapr/pkg/components/bindings"
	"github.com/dapr/dapr/pkg/components/pluggable"
	proto "github.com/dapr/dapr/pkg/proto/components/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type GRPCInputBinding struct {
	pluggable.GRPCComponent
	pluggable.InputBindingComponent
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
	protoMetadata := &proto.InitRequest{
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
	if result, err := ib.client.Read(context.TODO(), &emptypb.Empty{}); err != nil {
		return fmt.Errorf("unable to read from binding: %v", err)
	} else {
		r := b.ReadResponse{
			Data:     []byte{},
			Metadata: map[string]string{},
		}
		r.Data = result.Data
		r.Metadata = result.Metadata
		handler(context.TODO(), &r)
		return nil
	}
}

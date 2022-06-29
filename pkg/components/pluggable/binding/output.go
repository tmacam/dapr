package binding

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

type GRPCOutputBinding struct {
	pluggable.GRPCComponent
	pluggable.OutputBindingComponent
	name    string
	version string
	client  proto.OutputBindingClient
	context context.Context
}

func NewGRPCOutputBinding(name string, version string, socketPath string) (*GRPCOutputBinding, error) {
	socket := fmt.Sprintf("unix://%s", socketPath)
	if c, err := grpc.Dial(socket, grpc.WithInsecure()); err != nil {
		return nil, fmt.Errorf("unable to open GRPC connection using socket '%s': %v", socket, err)
	} else {
		return &GRPCOutputBinding{
			name:    name,
			version: version,
			client:  proto.NewOutputBindingClient(c),
			context: context.TODO(),
		}, nil
	}
}

func (ob *GRPCOutputBinding) Name() string {
	return ob.name
}

func (ob *GRPCOutputBinding) Version() string {
	return ob.version
}

func (ob *GRPCOutputBinding) OutputBinding() bindings.OutputBinding {
	return bindings.OutputBinding{
		Name: ob.Name(),
		FactoryMethod: func() b.OutputBinding {
			return ob
		},
	}
}

func (ob *GRPCOutputBinding) Close() error {
	return nil
}

func (ob *GRPCOutputBinding) Init(metadata b.Metadata) error {
	protoMetadata := &proto.MetadataRequest{
		Properties: map[string]string{},
	}
	for k, v := range metadata.Properties {
		protoMetadata.Properties[k] = v
	}

	_, err := ob.client.Init(context.TODO(), protoMetadata)
	return err
}

func (ob *GRPCOutputBinding) Ping() error {
	_, err := ob.client.Ping(ob.context, &emptypb.Empty{})
	return err
}

func (ob *GRPCOutputBinding) Invoke(ctx context.Context, req *b.InvokeRequest) (*b.InvokeResponse, error) {
	preq := proto.InvokeRequest{
		Data:      req.Data,
		Metadata:  req.Metadata,
		Operation: string(req.Operation),
	}

	if result, err := ob.client.Invoke(context.TODO(), &preq); err != nil {
		return nil, fmt.Errorf("unable to write to binding: %v", err)
	} else {
		r := b.InvokeResponse{
			Data:        result.Data,
			Metadata:    result.Metadata,
			ContentType: &result.Contenttype,
		}
		return &r, nil
	}
}

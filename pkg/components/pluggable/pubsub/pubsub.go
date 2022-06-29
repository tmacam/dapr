package state

import (
	"context"
	"fmt"
	"io"

	p "github.com/dapr/components-contrib/pubsub"
	"github.com/dapr/dapr/pkg/components/pluggable"
	"github.com/dapr/dapr/pkg/components/pubsub"
	proto "github.com/dapr/dapr/pkg/proto/components/v1"
	"github.com/dapr/kit/logger"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

var log = logger.NewLogger("grpcpubsub")

type GRPCPubSub struct {
	pluggable.GRPCComponent
	pluggable.PubSubComponent
	name     string
	version  string
	client   proto.PubSubClient
	features []p.Feature
	context  context.Context
}

func NewGRPCPubSub(name string, version string, socketPath string) (*GRPCPubSub, error) {
	socket := fmt.Sprintf("unix://%s", socketPath)
	if c, err := grpc.Dial(socket, grpc.WithInsecure()); err != nil {
		return nil, fmt.Errorf("unable to open GRPC connection using socket '%s': %v", socket, err)
	} else {
		return &GRPCPubSub{
			name:     name,
			version:  version,
			client:   proto.NewPubSubClient(c),
			features: []p.Feature{},
			context:  context.TODO(),
		}, nil
	}
}

func (ps *GRPCPubSub) Name() string {
	return ps.name
}

func (ps *GRPCPubSub) Version() string {
	return ps.version
}

func (ps *GRPCPubSub) PubSub() pubsub.PubSub {
	return pubsub.PubSub{
		Name: ps.Name(),
		FactoryMethod: func() p.PubSub {
			return ps
		},
	}
}

func (ps *GRPCPubSub) Close() error {
	return nil
}

func (ps *GRPCPubSub) Init(metadata p.Metadata) error {
	protoMetadata := &proto.MetadataRequest{
		Properties: map[string]string{},
	}
	for k, v := range metadata.Properties {
		protoMetadata.Properties[k] = v
	}

	// we need to call the method here because features could return an error and the features interface doesn't support errors
	featureResponse, err := ps.client.Features(context.TODO(), &emptypb.Empty{})
	if err != nil {
		return err
	}

	ps.features = []p.Feature{}
	for _, f := range featureResponse.Feature {
		feature := p.Feature(f)
		ps.features = append(ps.features, feature)
	}

	_, err = ps.client.Init(context.TODO(), protoMetadata)
	return err
}

func (ps *GRPCPubSub) Features() []p.Feature {
	return ps.features
}

func (ps *GRPCPubSub) Publish(req *p.PublishRequest) error {
	protoRequest := proto.PublishRequest{
		Topic:      req.Topic,
		Pubsubname: req.PubsubName,
		Data:       req.Data,
		Metadata:   req.Metadata,
	}
	_, err := ps.client.Publish(context.TODO(), &protoRequest)
	return err
}

func (ps *GRPCPubSub) Subscribe(req p.SubscribeRequest, handler p.Handler) error {
	protoRequest := proto.SubscribeRequest{
		Topic:    req.Topic,
		Metadata: req.Metadata,
	}
	stream, err := ps.client.Subscribe(context.Background(), &protoRequest)
	if err != nil {
		return fmt.Errorf("unable to subscribe: %v", err)
	}

	streamCtx := stream.Context()
	done := make(chan bool)

	// Read messages from the topic
	go func() error {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				close(done)
				return nil
			}
			if err != nil {
				return fmt.Errorf("failed to receive message: %v", err)
			}
			log.Debugf("Received message from stream on topic %s", resp.Topic)
			m := p.NewMessage{
				Data:        resp.Data,
				ContentType: &resp.Contenttype,
				Topic:       resp.Topic,
				Metadata:    resp.Metadata,
			}
			handler(streamCtx, &m)
		}
	}()

	return nil
}

func (ps *GRPCPubSub) Ping() error {
	/*	_, err := ps.client.Ping(ps.context, &emptypb.Empty{})
		return err
	*/
	return nil
}

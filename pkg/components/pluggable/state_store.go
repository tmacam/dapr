package pluggable

import (
	"context"
	"encoding/json"
	"fmt"

	s "github.com/dapr/components-contrib/state"
	"github.com/dapr/components-contrib/state/utils"
	"github.com/dapr/dapr/pkg/components/state"
	v1 "github.com/dapr/dapr/pkg/proto/common/v1"
	proto "github.com/dapr/dapr/pkg/proto/components/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type GRPCStateStore struct {
	GRPCComponent
	StateStoreComponent
	name     string
	version  string
	client   proto.StateStoreClient
	features []s.Feature
	context  context.Context
}

func NewGRPCStateStore(name string, version string, socketPath string) (*GRPCStateStore, error) {
	socket := fmt.Sprintf("unix://%s", socketPath)
	if c, err := grpc.Dial(socket, grpc.WithInsecure()); err != nil {
		return nil, fmt.Errorf("unable to open GRPC connection using socket '%s': %v", socket, err)
	} else {
		return &GRPCStateStore{
			name:     name,
			version:  version,
			client:   proto.NewStateStoreClient(c),
			features: []s.Feature{},
			context:  context.TODO(),
		}, nil
	}
}

func (ss *GRPCStateStore) Name() string {
	return ss.name
}

func (ss *GRPCStateStore) Version() string {
	return ss.version
}

func (ss *GRPCStateStore) StateStore() state.State {
	return state.State{
		Name: ss.Name(),
		FactoryMethod: func() s.Store {
			return ss
		},
	}
}

func (ss *GRPCStateStore) Close() error {
	return nil
}

func (ss *GRPCStateStore) Init(metadata s.Metadata) error {
	protoMetadata := &proto.MetadataRequest{
		Properties: map[string]string{},
	}
	for k, v := range metadata.Properties {
		protoMetadata.Properties[k] = v
	}

	// we need to call the method here because features could return an error and the features interface doesn't support errors
	featureResponse, err := ss.client.Features(context.TODO(), &emptypb.Empty{})
	if err != nil {
		return err
	}

	ss.features = []s.Feature{}
	for _, f := range featureResponse.Feature {
		feature := s.Feature(f)
		ss.features = append(ss.features, feature)
	}

	_, err = ss.client.Init(context.TODO(), protoMetadata)
	return err
}

func (ss *GRPCStateStore) Features() []s.Feature {
	return ss.features
}

func (ss *GRPCStateStore) Delete(req *s.DeleteRequest) error {
	_, err := ss.client.Delete(ss.context, &proto.DeleteRequest{
		Key: req.Key,
		Etag: &v1.Etag{
			Value: *req.ETag,
		},
		Metadata: req.Metadata,
		Options: &v1.StateOptions{
			Concurrency: getConcurrency(req.Options.Concurrency),
			Consistency: getConsistency(req.Options.Consistency),
		},
	})

	return err
}

func (ss *GRPCStateStore) Get(req *s.GetRequest) (*s.GetResponse, error) {
	etag := ""
	emptyResponse := &s.GetResponse{
		ETag:     &etag,
		Metadata: map[string]string{},
		Data:     []byte{},
	}

	response, err := ss.client.Get(context.TODO(), mapGetRequest(req))
	if err != nil {
		return emptyResponse, err
	}
	if response == nil {
		return emptyResponse, fmt.Errorf("response is nil")
	}

	return mapGetResponse(response), nil
}

func (ss *GRPCStateStore) Set(req *s.SetRequest) error {
	protoRequest, err := mapSetRequest(req)
	if err != nil {
		return err
	}
	_, err = ss.client.Set(context.TODO(), protoRequest)
	return err
}

func (ss *GRPCStateStore) Ping() error {
	_, err := ss.client.Ping(ss.context, &emptypb.Empty{})
	return err
}

func (ss *GRPCStateStore) BulkDelete(_ []s.DeleteRequest) error {
	return nil
}

func (ss *GRPCStateStore) BulkGet(req []s.GetRequest) (bool, []s.BulkGetResponse, error) {
	var protoRequests []*proto.GetRequest
	for _, request := range req {
		protoRequest := mapGetRequest(&request)
		protoRequests = append(protoRequests, protoRequest)
	}
	bulkGetRequest := &proto.BulkGetRequest{
		Items: protoRequests,
	}
	bulkGetResponse, err := ss.client.BulkGet(context.TODO(), bulkGetRequest)
	if err != nil {
		return false, nil, err
	}
	var items []s.BulkGetResponse
	for _, resp := range bulkGetResponse.Items {
		bulkGet := s.BulkGetResponse{
			Key:      resp.GetKey(),
			Data:     resp.GetData(),
			ETag:     &resp.GetEtag().Value,
			Metadata: resp.GetMetadata(),
			Error:    resp.Error,
		}
		items = append(items, bulkGet)
	}
	return bulkGetResponse.Got, items, nil
}

func (ss *GRPCStateStore) BulkSet(req []s.SetRequest) error {
	requests := []*proto.SetRequest{}
	for _, r := range req {
		protoRequest, err := mapSetRequest(&r)
		if err != nil {
			return err
		}
		requests = append(requests, protoRequest)
	}
	var err error
	_, err = ss.client.BulkSet(context.TODO(), &proto.BulkSetRequest{
		Items: requests,
	})
	return err
}

func mapSetRequest(req *s.SetRequest) (*proto.SetRequest, error) {
	var bytes []byte
	switch t := req.Value.(type) {
	case string:
		bytes = []byte(t)
	case []byte:
		bytes = t
	default:
		if t == nil {
			return nil, fmt.Errorf("set nil value")
		}
		var err error
		if bytes, err = utils.Marshal(t, json.Marshal); err != nil {
			return nil, err
		}
	}
	var etag *v1.Etag
	if req.ETag != nil {
		etag = &v1.Etag{
			Value: *req.ETag,
		}
	}
	return &proto.SetRequest{
		Key:      req.GetKey(),
		Value:    bytes,
		Etag:     etag,
		Metadata: req.GetMetadata(),
		Options: &v1.StateOptions{
			Concurrency: getConcurrency(req.Options.Concurrency),
			Consistency: getConsistency(req.Options.Consistency),
		},
	}, nil
}

func mapGetRequest(req *s.GetRequest) *proto.GetRequest {
	consistency, ok := v1.StateOptions_StateConsistency_value[req.Key]
	if !ok {
		consistency = int32(v1.StateOptions_CONSISTENCY_UNSPECIFIED)
	}
	return &proto.GetRequest{
		Key:         req.Key,
		Metadata:    req.Metadata,
		Consistency: v1.StateOptions_StateConsistency(consistency),
	}
}

func mapGetResponse(resp *proto.GetResponse) *s.GetResponse {
	var etag *string
	if resp.Etag != nil {
		etag = &resp.Etag.Value
	}
	return &s.GetResponse{
		Data:     resp.GetData(),
		ETag:     etag,
		Metadata: resp.GetMetadata(),
	}
}

func getConsistency(value string) v1.StateOptions_StateConsistency {
	consistency, ok := v1.StateOptions_StateConsistency_value[value]
	if !ok {
		return v1.StateOptions_CONSISTENCY_UNSPECIFIED
	}
	return v1.StateOptions_StateConsistency(consistency)
}

func getConcurrency(value string) v1.StateOptions_StateConcurrency {
	concurrency, ok := v1.StateOptions_StateConcurrency_value[value]
	if !ok {
		return v1.StateOptions_CONCURRENCY_UNSPECIFIED
	}
	return v1.StateOptions_StateConcurrency(concurrency)
}

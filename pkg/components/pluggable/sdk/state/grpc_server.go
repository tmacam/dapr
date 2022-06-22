package state

import (
	"context"
	"fmt"

	"github.com/dapr/components-contrib/state"
	"github.com/dapr/dapr/pkg/proto/common/v1"
	statev1pb "github.com/dapr/dapr/pkg/proto/components/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type GRPCStateStoreServer struct {
	statev1pb.StateStoreServer
	// this is the real implementation
	Impl state.Store
}

func (ss *GRPCStateStoreServer) Serve(srv *grpc.Server) error {
	statev1pb.RegisterStateStoreServer(srv, &GRPCStateStoreServer{Impl: ss.Impl})
	return nil
}

func (ss *GRPCStateStoreServer) Features(_ context.Context, _ *emptypb.Empty) (*statev1pb.FeaturesResponse, error) {
	features := ss.Impl.Features()
	var featureList []string
	for _, f := range features {
		featureList = append(featureList, string(f))
	}
	return &statev1pb.FeaturesResponse{
		Feature: featureList,
	}, nil
}

func (ss *GRPCStateStoreServer) Init(_ context.Context, req *statev1pb.MetadataRequest) (*emptypb.Empty, error) {
	metadata := state.Metadata{
		Properties: req.GetProperties(),
	}
	return &emptypb.Empty{}, ss.Impl.Init(metadata)
}

func (ss *GRPCStateStoreServer) Get(_ context.Context, req *statev1pb.GetRequest) (*statev1pb.GetResponse, error) {
	request := &state.GetRequest{
		Key:      req.GetKey(),
		Metadata: req.GetMetadata(),
		Options: state.GetStateOption{
			Consistency: req.Consistency.String(),
		},
	}
	response, err := ss.Impl.Get(request)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("response is nil")
	}
	var etag = &common.Etag{}
	if response.ETag != nil {
		etag.Value = *response.ETag
	}

	return &statev1pb.GetResponse{
		Data:     response.Data,
		Etag:     etag,
		Metadata: response.Metadata,
	}, nil
}

func (ss *GRPCStateStoreServer) Set(_ context.Context, req *statev1pb.SetRequest) (*emptypb.Empty, error) {
	var etag *string
	if req.Etag != nil {
		etag = &req.Etag.Value
	}
	setRequest := &state.SetRequest{
		Key:   req.Key,
		ETag:  etag,
		Value: req.Value,
		Options: state.SetStateOption{
			Concurrency: req.Options.GetConcurrency().String(),
			Consistency: req.Options.GetConsistency().String(),
		},
	}
	err := ss.Impl.Set(setRequest)
	return &emptypb.Empty{}, err
}

func (ss *GRPCStateStoreServer) Delete(_ context.Context, req *statev1pb.DeleteRequest) (*emptypb.Empty, error) {
	var etag *string
	if req.Etag != nil {
		etag = &req.Etag.Value
	}
	deleteRequest := &state.DeleteRequest{
		Key:      req.Key,
		ETag:     etag,
		Metadata: req.Metadata,
		Options: state.DeleteStateOption{
			Concurrency: req.Options.GetConcurrency().String(),
			Consistency: req.Options.GetConsistency().String(),
		},
	}
	return &emptypb.Empty{}, ss.Impl.Delete(deleteRequest)
}

func (ss *GRPCStateStoreServer) Ping(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, ss.Impl.Ping()
}

func (ss *GRPCStateStoreServer) BulkDelete(_ context.Context, _ *statev1pb.BulkDeleteRequest) (*emptypb.Empty, error) {
	return nil, nil
}

func (ss *GRPCStateStoreServer) BulkGet(_ context.Context, req *statev1pb.BulkGetRequest) (*statev1pb.BulkGetResponse, error) {
	var requests []state.GetRequest
	for _, protoRequest := range req.Items {
		stateRequest := state.GetRequest{
			Key:      protoRequest.GetKey(),
			Metadata: protoRequest.GetMetadata(),
			Options: state.GetStateOption{
				Consistency: protoRequest.Consistency.String(),
			},
		}
		requests = append(requests, stateRequest)
	}

	ok, responses, err := ss.Impl.BulkGet(requests)
	if err != nil {
		return nil, err
	}
	var items []*statev1pb.BulkStateItem
	for _, resp := range responses {
		var etag *common.Etag
		if resp.ETag != nil {
			etag = &common.Etag{
				Value: *resp.ETag,
			}
		}
		item := &statev1pb.BulkStateItem{
			Key:      resp.Key,
			Data:     resp.Data,
			Etag:     etag,
			Error:    resp.Error,
			Metadata: resp.Metadata,
		}
		items = append(items, item)
	}

	return &statev1pb.BulkGetResponse{
		Items: items,
		Got:   ok,
	}, nil
}

func (ss *GRPCStateStoreServer) mapSetRequest(stateSetRequest *statev1pb.SetRequest) *state.SetRequest {
	etag := ""
	if stateSetRequest.Etag != nil {
		etag = stateSetRequest.Etag.Value
	}
	return &state.SetRequest{
		Key:   stateSetRequest.Key,
		ETag:  &etag,
		Value: stateSetRequest.Value,
		Options: state.SetStateOption{
			Concurrency: stateSetRequest.Options.GetConcurrency().String(),
			Consistency: stateSetRequest.Options.GetConsistency().String(),
		},
	}
}
func (ss *GRPCStateStoreServer) BulkSet(_ context.Context, req *statev1pb.BulkSetRequest) (*emptypb.Empty, error) {
	var requests []state.SetRequest
	for _, protoSetRequest := range req.Items {
		stateSetRequest := ss.mapSetRequest(protoSetRequest)
		requests = append(requests, *stateSetRequest)
	}
	err := ss.Impl.BulkSet(requests)
	return &emptypb.Empty{}, err
}

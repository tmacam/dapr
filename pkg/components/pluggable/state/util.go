package state

import (
	"encoding/json"
	"fmt"
	s "github.com/dapr/components-contrib/state"
	"github.com/dapr/components-contrib/state/utils"
	v1 "github.com/dapr/dapr/pkg/proto/common/v1"
	proto "github.com/dapr/dapr/pkg/proto/components/v1"
)

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

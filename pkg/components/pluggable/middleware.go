package pluggable

import (
	"context"
	"fmt"

	"github.com/dapr/components-contrib/middleware"
	"github.com/dapr/dapr/pkg/components/middleware/http"
	http_middleware "github.com/dapr/dapr/pkg/middleware/http"
	proto "github.com/dapr/dapr/pkg/proto/components/v1"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type GRPCHttpMiddleware struct {
	GRPCComponent
	HttpMiddlewareComponent
	name     string
	version  string
	client   proto.HttpMiddlewareClient
	context  context.Context
	metadata middleware.Metadata
}

func NewGRPCHttpMiddleware(name string, version string, socketPath string) (*GRPCHttpMiddleware, error) {
	socket := fmt.Sprintf("unix://%s", socketPath)
	if c, err := grpc.Dial(socket, grpc.WithInsecure()); err != nil {
		return nil, fmt.Errorf("unable to open GRPC connection using socket '%s': %v", socket, err)
	} else {
		return &GRPCHttpMiddleware{
			name:    name,
			version: version,
			client:  proto.NewHttpMiddlewareClient(c),
			context: context.TODO(),
		}, nil
	}
}

func (m *GRPCHttpMiddleware) Name() string {
	return m.name
}

func (m *GRPCHttpMiddleware) Version() string {
	return m.version
}

func (m *GRPCHttpMiddleware) HttpMiddleware() http.Middleware {
	return http.Middleware{
		Name: m.Name(),
		FactoryMethod: func(metadata middleware.Metadata) (http_middleware.Middleware, error) {
			handler, err := m.GetHandler(metadata)
			return handler, err
		},
	}
}

func (m *GRPCHttpMiddleware) Close() error {
	return nil
}

func (m *GRPCHttpMiddleware) GetHandler(metadata middleware.Metadata) (func(h fasthttp.RequestHandler) fasthttp.RequestHandler, error) {
	log.Infof("HTTP middleware %s initializing...", m.Name)

	protoMetadata := &proto.MetadataRequest{
		Properties: map[string]string{},
	}

	for k, v := range metadata.Properties {
		protoMetadata.Properties[k] = v
	}

	handlerResponse, err := m.client.Init(context.TODO(), protoMetadata)

	if err != nil {
		return nil, err
	}

	return func(h fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			if handlerResponse.HeaderHandler {
				headers := make(map[string]string)

				ctx.Request.Header.VisitAllInOrder(func(key, value []byte) {
					headers[string(key[:])] = string(value[:])
				})

				protoRequest := &proto.HttpRequestHeader{
					Method:  string(ctx.Request.Header.Method()),
					Headers: headers,
				}

				if response, err := m.client.HandleHeader(context.TODO(), protoRequest); err != nil {
					log.Warnf("Error with component %s handling header", m.Name)
				} else {
					if response.RequestHeader != nil {
						log.Debugf("Updating request header with middleware request header")
						for k, v := range response.RequestHeader.Headers {
							ctx.Request.Header.Set(k, v)
						}

						ctx.Request.Header.SetMethod(response.RequestHeader.Method)
						ctx.Request.SetRequestURI(response.RequestHeader.Uri)
					} else if response.ResponseHeader != nil {
						log.Debugf("Updating response header with middleware response header")
						r := response.ResponseHeader
						ctx.Response.SetStatusCode(int(r.ResponseCode))
						for k, v := range r.Headers {
							ctx.Response.Header.Set(k, v)
						}

						if response.ResponseBody != nil {
							log.Debugf("Replacing response body with middleware response body")
							ctx.Response.SetBody(response.ResponseBody.Data)
						}
					}
				}

			}
			h(ctx)
		}
	}, nil

}

func (m *GRPCHttpMiddleware) Ping() error {
	_, err := m.client.Ping(m.context, &emptypb.Empty{})
	return err
}

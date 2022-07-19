## Quickstart

Prequisites for quickstart:

* All the required bits to compile `daprd` (go / make / standard build toolchain)
* Docker (to run the example pluggable component automatically)

Instructions:

1. Build `daprd` from this fork / branch (`make build` should be sufficient)
2. Start `daprd` pointing to the example component YAML for pluggable components (works out of the box, uses a public image whose source is available (on GitHub)[https://github.com/johnewart/dapr-memstore-go]:
```
./dist/<platform>/release/daprd --log-level debug --components-path ./tests/config/pluggable_components --app-id pluggable-test
```
3. Verify it works by writing / reading data from the in-memory data store directly using the daprd HTTP API:

```shell
curl -X POST -H "Content-Type: application/json" -d '[{ "key": "name", "value": "Bruce Wayne"}]' http://localhost:3500/v1.0/state/pluggablestate

curl http://localhost:3500/v1.0/state/pluggablestate/name
```
4. Clean up the docker container that was started (to be fixed soon)

## To create a pluggable component

The basic instructions are:

1. Generate code for the components protobuf files (in `dapr/proto/components/v1`)
2. Implement a gRPC service that handles these APIs
3. Serve the gRPC service on a UNIX domain socket 
   a. Manually register a component pointing to this domain socket
   b. Package your app as a container to auto-run the gRPC service
4. Create a `PluggableComponent` configuration to register your new component type
5. Instantiate a `Component` object using this newly created type 


(More docs to come, see https://github.com/johnewart/DaprPluggableComponentSDK.NET for an example of how to do this using C# / .NET)


## To hook up a pluggable component

Define a new pluggable component; the component either needs to be listening on a specific socket path, or run in a container. 

Example YAML for using a container (this will work out of the box, as this is a public image):

```yaml
apiVersion: dapr.io/v1alpha1
kind: PluggableComponent
metadata:
  name: memstore
spec:
  type: state
  version: v1
  container:
    image: johnewart/dapr-memstore-go
    version: latest
---
apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: pluggablestate
spec:
  type: state.memstore
  version: v1
```

For active development of a component, you can specify a socket path directly (make sure this socket exists before starting daprd or it will not be able to connect):

```yaml
apiVersion: dapr.io/v1alpha1
kind: PluggableComponent
metadata:
  name: dotnetredis
spec:
  type: state
  version: v1
  socket: /home/johnewart/Temp/sockets/dotnetcomponents.sock
---
apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: userstore
spec:
  type: state.dotnetredis
  version: v1
```

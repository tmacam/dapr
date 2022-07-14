## To create a pluggable component

More docs to come, see https://github.com/johnewart/DaprPluggableComponentSDK.NET for an example of how to do this using C# / .NET


## To use pluggable components

Define a new pluggable component; the component either needs to be listening on a specific socket path, or run in a container. 

Example YAML for using a container:

```yaml

apiVersion: dapr.io/v1alpha1
kind: PluggableComponent
metadata:
  name: memstore
spec:
  type: state
  version: v1
  container:
    image: dapr-memstore
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

Example for using a socket path:

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
---
```

package pluggable

type GRPCComponent interface {
	GRPCClient()
}

package jsonrpclite

type rpcServer struct {
	engine RpcServerEngine
}

//Start the server with the router.
func (server *rpcServer) Start(router *rpcRouter) {
	server.engine.Start(router)
}

//Stop the server
func (server *rpcServer) Stop() {
	server.engine.Stop()
}

// NewRpcServer Create a new rpc server with engine.
func NewRpcServer(engine RpcServerEngine) *rpcServer {
	server := new(rpcServer)
	server.engine = engine
	return server
}

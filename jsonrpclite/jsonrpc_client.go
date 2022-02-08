package jsonrpclite

type rpcClient struct {
	engine RpcClientEngine
}

// SendString Send request string to the server.
func (client *rpcClient) SendString(serviceName string, requestString string) string {
	return client.engine.ProcessString(serviceName, requestString)
}

// SendData Send request data to the server
func (client *rpcClient) SendData(serviceName string, method string, params []any) string {
	return client.engine.ProcessData(serviceName, method, params)
}

//Close the client if needed
func (client *rpcClient) Close() {
	client.engine.Close()
}

// NewRpcClient Create a new rpc client with engine.
func NewRpcClient(engine RpcClientEngine) *rpcClient {
	client := new(rpcClient)
	client.engine = engine
	return client
}

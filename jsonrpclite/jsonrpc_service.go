package jsonrpclite

import "errors"

type rpcService struct {
	name     string                //The name of the service
	instance any                   //The real instance of the service
	methods  map[string]*rpcMethod //Methods belong to this service

}

// addMethod Add method into the service
func (service *rpcService) addMethod(method *rpcMethod) {
	if service.methods == nil {
		service.methods = make(map[string]*rpcMethod)
	}
	service.methods[method.name] = method
}

// invoke call method of service by method name and parameter.
func (service *rpcService) invoke(request rpcRequest) rpcResponse {
	if service.methods == nil {
		var err any = errors.New("Can not find method " + request.method)
		panic(err)
	}
	paramCount := len(request.params)
	callParams := make([]any, paramCount+1)
	callParams[0] = service.instance
	for i := 0; i < paramCount; i++ {
		callParams[i+1] = request.params[i].value
	}
	method := service.methods[request.method]
	result := method.call(callParams)
	response := rpcResponse{id: request.id, isError: false, result: result}
	return response
}

// newRpcService Create a new rpcService
func newRpcService(name string, instance any) *rpcService {
	service := new(rpcService)
	service.instance = instance
	service.methods = make(map[string]*rpcMethod)
	service.name = name
	return service
}

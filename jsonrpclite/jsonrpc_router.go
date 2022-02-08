package jsonrpclite

import (
	"errors"
	"fmt"
	"reflect"
)

type rpcRouter struct {
	services map[string]*rpcService //Services in router
}

// NewRpcRouter Create a new rpc router.
func NewRpcRouter() *rpcRouter {
	return new(rpcRouter)
}

// dispatchRequests Dispatch request/s to services and get the response/s
func (router *rpcRouter) dispatchRequests(serviceName string, requests []rpcRequest) []rpcResponse {
	defer func() {
		var p = any(recover())
		if p != nil {
			errStr := fmt.Sprintln("Internal JSON-RPC error.") + fmt.Sprintf("%v", p)
			err := newRpcError(-32603, errStr)
			response := rpcResponse{-1, true, err}
			var responseErr any = newRpcResponseError(response)
			panic(responseErr)
		}
	}()
	rpcService := router.getService(serviceName)
	if rpcService == nil {
		var err any = errors.New("Service " + serviceName + " does not exist.")
		panic(err)
	} else {
		if len(requests) > 1 {
			var responses = make([]rpcResponse, 0)
			for i := 0; i < len(requests); i++ {
				request := requests[i]
				response := rpcService.invoke(request)
				if !request.isNotification() {
					responses = append(responses, rpcResponse{request.id, false, response.result})
				}
			}
			return responses
		} else {
			var responses = make([]rpcResponse, 0)
			request := requests[0]
			response := rpcService.invoke(request)
			if !request.isNotification() {
				responses = append(responses, response)
			}
			return responses
		}
	}
}

//Get the service by service name
func (router *rpcRouter) getService(serviceName string) *rpcService {
	if router.services == nil {
		return nil
	}
	return router.services[serviceName]
}

// RegisterService Register the logic service into the router
func (router *rpcRouter) RegisterService(serviceName string, serviceInstance any) {
	instanceType := reflect.TypeOf(serviceInstance)
	s := newRpcService(serviceName, serviceInstance)
	numMethod := instanceType.NumMethod()
	if numMethod > 0 {
		for i := 0; i < numMethod; i++ {
			method := instanceType.Method(i)
			if method.IsExported() {
				instanceMethod, ok := instanceType.MethodByName(method.Name)
				if ok {
					rpcMethod := newRpcMethod(instanceMethod)
					s.addMethod(rpcMethod)
				} else {
					var err any = errors.New("The output param count of" + instanceType.Name() + "." + instanceMethod.Name + " is not matched")
					panic(err)
				}
			}
		}
	}
	if router.services == nil {
		router.services = make(map[string]*rpcService)
	}
	router.services[serviceName] = s
}

package jsonrpclite

import (
	"errors"
	"reflect"
)

type rpcMethodType uint32

const (
	returnMethod rpcMethodType = iota //Handler with return value
	voidMethod                        //Handler without return value
)

type rpcMethodHandler func(params []any) any

type rpcMethod struct {
	handler    rpcMethodHandler //The method reflect value
	name       string           //The name of the method
	methodType rpcMethodType    //The type of the handler
	paramTypes []reflect.Type   //The types of the handler params
	returnType reflect.Type     //The type of return value
}

//call the method of the rpcMethod
func (method *rpcMethod) call(params []any) any {
	return method.handler(params)
}

// newRpcMethod Create a new rpcMethod instance
func newRpcMethod(serviceMethod reflect.Method) *rpcMethod {
	//Parse out
	outNum := serviceMethod.Type.NumOut()
	if outNum > 1 {
		var err any = errors.New("The return value count of method " + serviceMethod.Name + " should be 0 or 1")
		panic(err)
	}
	//Parse in
	inNum := serviceMethod.Type.NumIn()
	paramTypes := make([]reflect.Type, inNum)
	for i := 0; i < inNum; i++ {
		paramTypes[i] = serviceMethod.Type.In(i)
	}
	method := new(rpcMethod)
	method.name = serviceMethod.Name
	method.paramTypes = paramTypes
	if outNum == 0 {
		method.methodType = voidMethod
		method.returnType = reflect.TypeOf(nil)
		method.handler = func(params []any) any {
			paramCount := len(params)
			callParams := make([]reflect.Value, paramCount)
			for i := 0; i < paramCount; i++ {
				callParams[i] = reflect.ValueOf(params[i])
			}
			serviceMethod.Func.Call(callParams)
			return nil
		}
	} else {
		method.methodType = returnMethod
		method.returnType = serviceMethod.Type.Out(0)
		method.handler = func(params []any) any {
			paramCount := len(params)
			callParams := make([]reflect.Value, paramCount)
			for i := 0; i < paramCount; i++ {
				callParams[i] = reflect.ValueOf(params[i])
			}
			result := serviceMethod.Func.Call(callParams)[0]
			return result.Interface()
		}
	}
	return method
}

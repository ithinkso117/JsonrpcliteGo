package jsonrpclite

import "reflect"

type rpcParam struct {
	_type reflect.Type //The type of the param
	value any          //The value of the param.
}

type rpcRequest struct {
	id     any        //The id of the request
	method string     //The method name of the request
	params []rpcParam // params stored in this request
}

// isNotification Check whether the request is a notification.
func (request rpcRequest) isNotification() bool {
	return request.id == nil
}

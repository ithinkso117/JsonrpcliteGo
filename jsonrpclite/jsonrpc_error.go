package jsonrpclite

import (
	"fmt"
)

type rpcError struct {
	Code    int    `json:"Code"`
	Message string `json:"Message"`
}

func (err *rpcError) Error() string {
	return "rpcError:" + fmt.Sprintf("%v", err.Code) + "; " + err.Message
}

// newRpcError Create a RpcError
func newRpcError(code int, msg string) error {
	err := new(rpcError)
	err.Code = code
	err.Message = msg
	return err
}

// RpcResponseError An error which contains the response string.
type RpcResponseError struct {
	response string
}

func (responseError *RpcResponseError) Error() string {
	return "RpcErrors:" + responseError.response
}

func newRpcResponseError(response rpcResponse) *RpcResponseError {
	err := new(RpcResponseError)
	err.response = string(encodeResponses([]rpcResponse{response}))
	return err
}

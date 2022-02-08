package jsonrpclite

type rpcResponse struct {
	id      any  //The id of the response which was from the request.
	isError bool //True when the result is an error, otherwise is the result.
	result  any  //The result or error of the response.
}

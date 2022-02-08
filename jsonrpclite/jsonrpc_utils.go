package jsonrpclite

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type requestData struct {
	JsonRpc string          `json:"jsonrpc"`
	Id      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

func decodeRequest(service *rpcService, data requestData) rpcRequest {
	defer func() {
		var p = any(recover())
		if p != nil {
			responseErr, ok := p.(*RpcResponseError)
			if ok {
				panic(any(responseErr))
			} else {
				errStr := fmt.Sprintln("The JSON sent is not a valid Request object.") + fmt.Sprintf("%v", p)
				err := newRpcError(-32600, errStr)
				response := rpcResponse{-1, true, err}
				var responseErr any = newRpcResponseError(response)
				panic(responseErr)
			}
		}
	}()
	method := service.methods[data.Method]
	if method == nil {
		errStr := "The method does not exist / is not available."
		err := newRpcError(-32601, errStr)
		response := rpcResponse{-1, true, err}
		var responseErr any = newRpcResponseError(response)
		panic(responseErr)
	}
	request := rpcRequest{id: data.Id, method: data.Method}
	paramTypes := method.paramTypes
	paramCount := len(paramTypes)
	switch paramCount {
	case 0:
		errStr := fmt.Sprintln("Invalid method parameter(s).") + "method " + data.Method + "'s param count should contains receiver."
		err := newRpcError(-32601, errStr)
		response := rpcResponse{-1, true, err}
		var responseErr any = newRpcResponseError(response)
		panic(responseErr)
	case 1:
		if data.Params != nil {
			errStr := fmt.Sprintln("Invalid method parameter(s).") + "method " + data.Method + "'s param count should be 0."
			err := newRpcError(-32601, errStr)
			response := rpcResponse{-1, true, err}
			var responseErr any = newRpcResponseError(response)
			panic(responseErr)
		}
		request.params = make([]rpcParam, 0)
	case 2:
		if data.Params == nil {
			errStr := fmt.Sprintln("Invalid method parameter(s).") + "Param of method " + data.Method + "is empty."
			err := newRpcError(-32601, errStr)
			response := rpcResponse{-1, true, err}
			var responseErr any = newRpcResponseError(response)
			panic(responseErr)
		}
		paramStr := string(data.Params)
		isArray := len(paramStr) > 0 && paramStr[0] == '['
		if isArray {
			//Array is not matched
			errStr := fmt.Sprintln("Invalid method parameter(s).") + "Param count of method" + data.Method + " is not matched."
			err := newRpcError(-32601, errStr)
			response := rpcResponse{-1, true, err}
			var responseErr any = newRpcResponseError(response)
			panic(responseErr)
		} else {
			rpcParamValue := reflect.New(method.paramTypes[1]).Interface()
			err := json.Unmarshal(data.Params, &rpcParamValue)
			if err == nil {
				request.params = []rpcParam{{method.paramTypes[1], rpcParamValue}}
			} else {
				errStr := fmt.Sprintln("Invalid method parameter(s).") + "UnMarshal param error:" + err.Error()
				err := newRpcError(-32601, errStr)
				response := rpcResponse{-1, true, err}
				var responseErr any = newRpcResponseError(response)
				panic(responseErr)
			}
		}
	default:
		//Array
		paramValues := make([]any, paramCount-1)
		for i := 0; i < paramCount-1; i++ {
			paramType := paramTypes[i+1]
			paramValue := reflect.New(paramType).Interface()
			paramValues[i] = paramValue
		}
		err := json.Unmarshal(data.Params, &paramValues)
		if err == nil {
			rpcParams := make([]rpcParam, paramCount-1)
			for i := 0; i < paramCount-1; i++ {
				rpcParams[i] = rpcParam{method.paramTypes[i+1], reflect.ValueOf(paramValues[i]).Elem().Interface()}
			}
			request.params = rpcParams
		} else {
			errStr := fmt.Sprintln("Invalid method parameter(s).") + "UnMarshal param error:" + err.Error()
			err := newRpcError(-32601, errStr)
			response := rpcResponse{-1, true, err}
			var responseErr any = newRpcResponseError(response)
			panic(responseErr)
		}
	}
	return request
}

func decodeRequestString(service *rpcService, jsonStr string) []rpcRequest {
	defer func() {
		var p = any(recover())
		if p != nil {
			responseErr, ok := p.(*RpcResponseError)
			if ok {
				panic(any(responseErr))
			} else {
				errStr := fmt.Sprintln("Invalid JSON was received by the server. An error occurred on the server while parsing the JSON text.") + fmt.Sprintf("%v", p)
				err := newRpcError(-32700, errStr)
				response := rpcResponse{-1, true, err}
				var responseErr any = newRpcResponseError(response)
				panic(responseErr)
			}
		}
	}()
	isArray := len(jsonStr) > 0 && jsonStr[0] == '['
	if isArray {
		var requestsData []requestData
		err := json.Unmarshal([]byte(jsonStr), &requestsData)
		if err == nil {
			requests := make([]rpcRequest, len(requestsData))
			for i := 0; i < len(requestsData); i++ {
				requests[i] = decodeRequest(service, requestsData[i])
			}
			return requests
		} else {
			panic(any(err))
		}

	} else {
		var requestData = new(requestData)
		err := json.Unmarshal([]byte(jsonStr), &requestData)
		if err == nil {
			request := decodeRequest(service, *requestData)
			return []rpcRequest{request}
		} else {
			panic(any(err))
		}
	}
	return nil
}

func createResponseData(response rpcResponse) map[string]any {
	result := make(map[string]any)
	result["jsonrpc"] = "2.0"
	result["id"] = response.id
	switch response.result.(type) {
	case *rpcError:
		result["error"] = response.result
	default:
		result["result"] = response.result
	}
	return result
}

func encodeResponses(responses []rpcResponse) []byte {
	if len(responses) == 0 {
		return nil
	}
	if len(responses) == 1 {
		response := responses[0]
		result := createResponseData(response)
		buffer, err := json.Marshal(result)
		if err == nil {
			return buffer
		}
		var jsonError any = err
		panic(jsonError)
	} else {
		numResponses := len(responses)
		results := make([]map[string]any, numResponses)
		for i := 0; i < numResponses; i++ {
			response := responses[i]
			result := createResponseData(response)
			results[i] = result
		}
		buffer, err := json.Marshal(results)
		if err == nil {
			return buffer
		}
		var jsonError any = err
		panic(jsonError)
	}
}

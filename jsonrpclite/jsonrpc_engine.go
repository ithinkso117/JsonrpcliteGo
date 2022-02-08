package jsonrpclite

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type RpcServerEngine interface {
	// GetName Get the engine name.
	GetName() string
	//Start the engine and initialize the router.
	Start(router *rpcRouter)
	//Stop the engine and free the router.
	Stop()
}

type RpcClientEngine interface {
	// GetName Get the engine name.
	GetName() string
	// ProcessString Send the rpc request string to the server.
	ProcessString(serviceName string, requestStr string) string
	// ProcessData Send the rpc request data to the server
	ProcessData(serviceName string, method string, params []any) string
	//Close the engine and free the router.
	Close()
}

// RpcServerEngineCore The basic server engine for other engines
type RpcServerEngineCore struct {
	_router *rpcRouter
}

// SetRouter Initialize the router for the engine.
func (engine *RpcServerEngineCore) SetRouter(router *rpcRouter) {
	engine._router = router
}

// ServiceExists check whether the service is available.
func (engine *RpcServerEngineCore) ServiceExists(serviceName string) bool {
	return engine._router.getService(serviceName) != nil
}

// Dispatch the request string to the services.
func (engine *RpcServerEngineCore) Dispatch(serviceName string, requestStr string) string {
	if engine._router == nil {
		var err any = errors.New(" The rpc router has not been initialized. ")
		panic(err)
	}
	service := engine._router.getService(serviceName)
	if service != nil {
		requests := decodeRequestString(service, requestStr)
		responses := engine._router.dispatchRequests(serviceName, requests)
		if len(responses) > 0 {
			result := encodeResponses(responses)
			return string(result)
		} else {
			return ""
		}
	} else {
		var err any = errors.New("Service " + serviceName + " does not exist.")
		panic(err)
	}
}

//The engine for in-process communication
type rpcInProcessEngine struct {
	requestId int
	*RpcServerEngineCore
}

// GetName Get the engine name.
func (engine *rpcInProcessEngine) GetName() string {
	return "RpcInProcessEngine"
}

//Start the engine and initialize the router.
func (engine *rpcInProcessEngine) Start(router *rpcRouter) {
	engine.RpcServerEngineCore.SetRouter(router)
}

//Stop the engine and free the router.
func (engine *rpcInProcessEngine) Stop() {
	engine.RpcServerEngineCore.SetRouter(nil)
}

// ProcessString Send the rpc request string to the server.
func (engine *rpcInProcessEngine) ProcessString(serviceName string, requestStr string) string {
	return engine.RpcServerEngineCore.Dispatch(serviceName, requestStr)
}

// ProcessData Send the rpc request data to the server.
func (engine *rpcInProcessEngine) ProcessData(serviceName string, method string, params []any) string {
	engine.requestId++
	data := requestData{"2.0", engine.requestId, method, nil}
	if len(params) > 0 {
		if len(params) == 1 {
			paramData, err := json.Marshal(params[0])
			if err == nil {
				data.Params = paramData
			} else {
				var sendErr any = errors.New("Send request fail: " + err.Error())
				panic(sendErr)
			}
		} else {
			paramsData, err := json.Marshal(params)
			if err == nil {
				data.Params = paramsData
			} else {
				var sendErr any = errors.New("Send request fail: " + err.Error())
				panic(sendErr)
			}
		}
	}
	requestData, err := json.Marshal(data)
	if err == nil {
		return engine.RpcServerEngineCore.Dispatch(serviceName, string(requestData))
	} else {
		var sendErr any = errors.New("Send request fail: " + err.Error())
		panic(sendErr)
	}
}

//Close the engine and free the router.
func (engine *rpcInProcessEngine) Close() {
	engine.RpcServerEngineCore.SetRouter(nil)
}

// NewInProcessEngine Create an InProcessEngine, the results are the same instance.
func NewInProcessEngine() (RpcServerEngine, RpcClientEngine) {
	engine := new(rpcInProcessEngine)
	engine.RpcServerEngineCore = new(RpcServerEngineCore)
	return engine, engine
}

//A basic http server engine which uses the build-in http lib.
type rpcHttpServerEngine struct {
	port int
	*RpcServerEngineCore
}

// GetName Get the engine name.
func (engine *rpcHttpServerEngine) GetName() string {
	return "RpcHttpServerEngine"
}

//Start the engine and initialize the router.
func (engine *rpcHttpServerEngine) Start(router *rpcRouter) {
	engine.RpcServerEngineCore.SetRouter(router)
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		defer func() {
			var p = any(recover())
			if p != nil {
				responseErr, ok := p.(*RpcResponseError)
				if ok {
					errData := []byte(responseErr.response)
					writer.Header().Set("Server", "JsonRpcLite-Go")
					writer.Header().Set("Access-Control-Allow-Origin", "*")
					writer.Header().Set("Content-Type", "application/json; charset=utf-8")
					writer.Header().Set("Content-Length", strconv.Itoa(len(errData)))
					writer.WriteHeader(http.StatusOK)
					_, err := writer.Write(errData)
					if err != nil {
						logger.Warning("Write data to client failed: " + err.Error())
					}
				} else {
					//Unhandled system error
					errStr := "Server error: " + fmt.Sprintf("%v", p)
					errData := []byte(errStr)
					writer.Header().Set("Server", "JsonRpcLite-Go")
					writer.Header().Set("Access-Control-Allow-Origin", "*")
					writer.Header().Set("Content-Type", "text/html; charset=utf-8")
					writer.Header().Set("Content-Length", strconv.Itoa(len(errData)))
					writer.WriteHeader(http.StatusInternalServerError)
					_, err := writer.Write(errData)
					if err != nil {
						logger.Warning("Write data to client failed: " + err.Error())
					}
				}
			}
		}()
		serviceName := strings.Replace(request.URL.Path, "/", "", -1)
		if request.Method == "POST" {
			contentLength := request.ContentLength
			buffer := new(bytes.Buffer)
			for contentLength > 0 {
				size, err := buffer.ReadFrom(request.Body)
				if err == nil {
					contentLength -= size
				} else {
					break
				}
			}
			if engine.ServiceExists(serviceName) {
				response := engine.RpcServerEngineCore.Dispatch(serviceName, string(buffer.Bytes()))
				if response != "" {
					responseData := []byte(response)
					writer.Header().Set("Server", "JsonRpcLite-Go")
					writer.Header().Set("Access-Control-Allow-Origin", "*")
					writer.Header().Set("Content-Type", "application/json; charset=utf-8")
					writer.Header().Set("Content-Length", strconv.Itoa(len(responseData)))
					writer.WriteHeader(http.StatusOK)
					_, err := writer.Write(responseData)
					if err != nil {
						logger.Warning("Write data to client failed: " + err.Error())
					}
				} else {
					writer.Header().Set("Server", "JsonRpcLite-Go")
					writer.Header().Set("Access-Control-Allow-Origin", "*")
					writer.Header().Set("Content-Length", "0")
					writer.WriteHeader(http.StatusOK)
				}
			} else {
				errStr := "Service " + serviceName + " does not exist."
				errData := []byte(errStr)
				writer.Header().Set("Server", "JsonRpcLite-Go")
				writer.Header().Set("Access-Control-Allow-Origin", "*")
				writer.Header().Set("Content-Type", "text/html; charset=utf-8")
				writer.Header().Set("Content-Length", strconv.Itoa(len(errData)))
				writer.WriteHeader(http.StatusServiceUnavailable)
				_, err := writer.Write(errData)
				if err != nil {
					logger.Warning("Write data to client failed: " + err.Error())
				}
			}
		} else {
			errStr := "Invalid http-method: " + request.Method
			errData := []byte(errStr)
			writer.Header().Set("Server", "JsonRpcLite-Go")
			writer.Header().Set("Access-Control-Allow-Origin", "*")
			writer.Header().Set("Content-Type", "text/html; charset=utf-8")
			writer.Header().Set("Content-Length", strconv.Itoa(len(errData)))
			writer.WriteHeader(http.StatusMethodNotAllowed)
			_, err := writer.Write(errData)
			if err != nil {
				logger.Warning("Write data to client failed: " + err.Error())
			}
		}
	})
	go func() {
		logger.Info(http.ListenAndServe(":"+strconv.Itoa(engine.port), nil).Error())
	}()
}

//Stop the engine and free the router.
func (engine *rpcHttpServerEngine) Stop() {
	engine.RpcServerEngineCore.SetRouter(nil)
}

// NewRpcHttpServerEngine Create a new RpcServerHttpEngine which based on the build-in http lib
func NewRpcHttpServerEngine(port int) RpcServerEngine {
	engine := new(rpcHttpServerEngine)
	engine.port = port
	engine.RpcServerEngineCore = new(RpcServerEngineCore)
	return engine
}

//A basic http client engine which uses the build-in http lib.
type rpcHttpClientEngine struct {
	requestId  int
	serverHost string
}

// GetName Get the engine name.
func (engine *rpcHttpClientEngine) GetName() string {
	return "RpcHttpClientEngine"
}

// ProcessData Send the rpc request data to the server.
func (engine *rpcHttpClientEngine) ProcessData(serviceName string, method string, params []any) string {
	engine.requestId++
	data := requestData{"2.0", engine.requestId, method, nil}
	if len(params) > 0 {
		if len(params) == 1 {
			paramData, err := json.Marshal(params[0])
			if err == nil {
				data.Params = paramData
			} else {
				var sendErr any = errors.New("Send request fail: " + err.Error())
				panic(sendErr)
			}
		} else {
			paramsData, err := json.Marshal(params)
			if err == nil {
				data.Params = paramsData
			} else {
				var sendErr any = errors.New("Send request fail: " + err.Error())
				panic(sendErr)
			}
		}
	}
	requestData, err := json.Marshal(data)
	if err == nil {
		return engine.ProcessString(serviceName, string(requestData))
	} else {
		var sendErr any = errors.New("Send request fail: " + err.Error())
		panic(sendErr)
	}
}

// ProcessString Process Send the rpc request to the server.
func (engine *rpcHttpClientEngine) ProcessString(serviceName string, requestStr string) string {
	buffer := new(bytes.Buffer)
	_, err := buffer.WriteString(requestStr)
	if err == nil {
		response, err := http.Post(engine.serverHost+"/"+serviceName, "application/json; charset=utf-8", buffer)
		if err == nil {
			contentLength := response.ContentLength
			buffer = new(bytes.Buffer)
			for contentLength > 0 {
				size, err := buffer.ReadFrom(response.Body)
				if err == nil {
					contentLength -= size
				} else {
					break
				}
			}
			return string(buffer.Bytes())
		} else {
			var sendErr any = errors.New("Send request fail: " + err.Error())
			panic(sendErr)
		}
	} else {
		var sendErr any = errors.New("Send request fail: " + err.Error())
		panic(sendErr)
	}
}

//Close the engine and free the router.
func (engine *rpcHttpClientEngine) Close() {
	//DoNothing
}

// NewRpcHttpClientEngine Create a new rpc http client based on the build-in http lib
func NewRpcHttpClientEngine(serverHost string) RpcClientEngine {
	engine := new(rpcHttpClientEngine)
	engine.serverHost = serverHost
	return engine
}

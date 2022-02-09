package jsonrpclite

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
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
				var sendErr any = errors.New("Send request error: " + err.Error())
				panic(sendErr)
			}
		} else {
			paramsData, err := json.Marshal(params)
			if err == nil {
				data.Params = paramsData
			} else {
				var sendErr any = errors.New("Send request error: " + err.Error())
				panic(sendErr)
			}
		}
	}
	requestData, err := json.Marshal(data)
	if err == nil {
		return engine.RpcServerEngineCore.Dispatch(serviceName, string(requestData))
	} else {
		var sendErr any = errors.New("Send request error: " + err.Error())
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

type rpcHttpServerHandler struct {
	engine *rpcHttpServerEngine
}

func (handler *rpcHttpServerHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	engine := handler.engine
	defer func() {
		var p = any(recover())
		if p != nil {
			responseErr, ok := p.(*RpcResponseError)
			if ok {
				engine.WriteResponseData(writer, http.StatusOK, "application/json", responseErr.response)
			} else {
				//Unhandled system error
				errStr := "Server error: " + fmt.Sprintf("%v", p)
				engine.WriteResponseData(writer, http.StatusInternalServerError, "text/html", errStr)
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
				engine.WriteResponseData(writer, http.StatusOK, "application/json", response)
			} else {
				engine.WriteResponseData(writer, http.StatusOK, "", "")
			}
		} else {
			errStr := "Service " + serviceName + " does not exist."
			engine.WriteResponseData(writer, http.StatusServiceUnavailable, "text/html", errStr)
		}
	} else {
		errStr := "Invalid http-method: " + request.Method
		engine.WriteResponseData(writer, http.StatusMethodNotAllowed, "text/html", errStr)
	}
}

//A basic http server engine which uses the build-in http lib.
type rpcHttpServerEngine struct {
	server *http.Server
	port   int
	*RpcServerEngineCore
}

// GetName Get the engine name.
func (engine *rpcHttpServerEngine) GetName() string {
	return "RpcHttpServerEngine"
}

// WriteResponseData Common way to write string data to client.
func (engine *rpcHttpServerEngine) WriteResponseData(writer http.ResponseWriter, statusCode int, contentType string, content string) {
	contentData := []byte(content)
	writer.Header().Set("Server", "JsonRpcLite-Go")
	writer.Header().Set("Access-Control-Allow-Origin", "*")
	if contentType != "" {
		writer.Header().Set("Content-Type", contentType+"; charset=utf-8")
	}
	if len(contentData) > 0 {
		writer.Header().Set("Content-Length", strconv.Itoa(len(contentData)))
	}
	writer.WriteHeader(statusCode)
	if len(contentData) > 0 {
		_, err := writer.Write(contentData)
		if err != nil {
			logger.Warning("Write data to client error: " + err.Error())
		}
	}
}

//Start the engine and initialize the router.
func (engine *rpcHttpServerEngine) Start(router *rpcRouter) {
	if engine.server != nil {
		logger.Warning("The server of engine already started, will be closed.")
		engine.Stop()
	}
	server := new(http.Server)
	engine.RpcServerEngineCore.SetRouter(router)
	handler := new(rpcHttpServerHandler)
	handler.engine = engine
	server.Handler = handler
	server.Addr = ":" + strconv.Itoa(engine.port)
	go func() {
		logger.Info(server.ListenAndServe().Error())
	}()
}

//Stop the engine and free the router.
func (engine *rpcHttpServerEngine) Stop() {
	if engine.server != nil {
		err := engine.server.Close()
		if err != nil {
			logger.Warning("Close the server of engine error: " + err.Error())
		}
		engine.server = nil
	}
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
				var sendErr any = errors.New("Send request error: " + err.Error())
				panic(sendErr)
			}
		} else {
			paramsData, err := json.Marshal(params)
			if err == nil {
				data.Params = paramsData
			} else {
				var sendErr any = errors.New("Send request error: " + err.Error())
				panic(sendErr)
			}
		}
	}
	requestData, err := json.Marshal(data)
	if err == nil {
		return engine.ProcessString(serviceName, string(requestData))
	} else {
		var sendErr any = errors.New("Send request error: " + err.Error())
		panic(sendErr)
	}
}

// ProcessString Process Send the rpc request to the server.
func (engine *rpcHttpClientEngine) ProcessString(serviceName string, requestStr string) string {
	buffer := bytes.NewBufferString(requestStr)
	http.DefaultClient.Timeout = 5 * time.Second
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
		var sendErr any = errors.New("Send request error: " + err.Error())
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

package main

import (
	"fmt"
	"jsonrpclite/jsonrpclite"
	"strconv"
)

type ParamData1 struct {
	A int
	B []string
}

type ParamData2 struct {
	A int
	B string
}

type ITest interface {
	MyTest(arg1 string, arg2 int, arg3 ParamData1, arg4 ParamData2) string
}

type TestService struct {
}

func (test *TestService) MyTest(arg1 string, arg2 int, arg3 ParamData1, arg4 ParamData2) string {
	return arg1 + " " + strconv.Itoa(arg2) + " " + strconv.Itoa(arg3.A) + " " + arg4.B + " from Test1"
}

func main() {
	router := jsonrpclite.NewRpcRouter()
	serviceInstance := new(TestService)
	router.RegisterService("ITest", serviceInstance)
	serverEngine := jsonrpclite.NewRpcHttpServerEngine(8080)
	server := jsonrpclite.NewRpcServer(serverEngine)
	server.Start(router)
	fmt.Println("Rpc server started.")

	clientEngine := jsonrpclite.NewRpcHttpClientEngine("http://localhost:8080")
	client := jsonrpclite.NewRpcClient(clientEngine)
	result := client.SendString("ITest", "{\n    \"jsonrpc\": \"2.0\",\n    \"id\": 1,\n    \"method\": \"MyTest\",\n    \"params\": [\n      \"aaaaaaaa\",\n      10,\n      {\n        \"a\": 12,\n        \"b\": [ \"test\", \"test\", \"test\" ]\n      },\n      {\n        \"a\": 12,\n        \"b\": \"test\"\n      }\n    ]\n  }")
	fmt.Print(result)

	fmt.Scanln()
	server.Stop()
	fmt.Println("Rpc server stopped..")
}

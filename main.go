package main

import (
	"fmt"
	"jsonrpclite/jsonrpclite"
	"strconv"
	"strings"
)

type ParamData1 struct {
	A int
	B []string
}

type ParamData2 struct {
	A int
	B string
}

type Result struct {
	A string
	B string
	C string
	D string
}

type TestService struct {
}

func (test *TestService) MyTest(arg1 string, arg2 int, arg3 ParamData1, arg4 ParamData2) Result {
	return Result{arg1, strconv.Itoa(arg2), "[" + strconv.Itoa(arg3.A) + "]" + strings.Join(arg3.B, ","), "[" + strconv.Itoa(arg4.A) + "]" + arg4.B}
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
	for i := 0; i < 100; i++ {
		result := client.SendData("ITest", "MyTest", []any{"Hello", 999, ParamData1{666, []string{"你好", "世界"}}, ParamData2{555, "甜蜜的世界"}})
		fmt.Print(result)
	}
	fmt.Scanln()
	server.Stop()
	fmt.Println("Rpc server stopped..")
}

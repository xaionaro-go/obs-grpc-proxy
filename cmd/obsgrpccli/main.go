package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/facebookincubator/go-belt/tool/logger"
	xlogrus "github.com/facebookincubator/go-belt/tool/logger/implementation/logrus"
	"github.com/spf13/pflag"
	"github.com/xaionaro-go/obs-grpc-proxy/protobuf/go/obs_grpc"
	"google.golang.org/grpc"
)

func assertNoError(ctx context.Context, err error) {
	if err != nil {
		logger.Panic(ctx, err)
	}
}

func main() {
	clientSampleV := reflect.TypeOf(obs_grpc.NewOBSClient(nil))
	var methods []string
	for i := 0; i < clientSampleV.NumMethod(); i++ {
		method := clientSampleV.Method(i)
		methods = append(methods, method.Name)
	}

	logLevel := logger.LevelInfo
	pflag.Var(&logLevel, "log-level", "Log level")
	grpcProxyAddr := pflag.String("grpc-proxy-addr", "localhost:4456", "the address of the OBS gRPC proxy")
	methodName := pflag.String("method-name", "", fmt.Sprintf("available values: %s", strings.Join(methods, ", ")))
	data := pflag.String("request-data", "", "the JSON of the data to be sent to the server")
	pflag.Parse()

	ctx := context.Background()
	l := xlogrus.Default().WithLevel(logLevel)
	ctx = logger.CtxWithLogger(ctx, l)

	conn, err := grpc.NewClient(*grpcProxyAddr, grpc.WithInsecure())
	assertNoError(ctx, err)

	client := obs_grpc.NewOBSClient(conn)
	clientT := reflect.TypeOf(client)
	clientV := reflect.ValueOf(client)

	methodV := clientV.MethodByName(*methodName)
	methodT, ok := clientT.MethodByName(*methodName)
	if !ok {
		panic(fmt.Errorf("method '%s' not found, available methods: %s", *methodName, strings.Join(methods, ", ")))
	}
	inputT := methodT.Type.In(2)
	inputV := reflect.New(inputT.Elem()).Elem()
	err = json.Unmarshal([]byte(*data), inputV.Addr().Interface())
	if err != nil {
		panic(fmt.Errorf("unable to unserialize the input to %T: %w", inputV.Interface(), err))
	}

	result := methodV.Call(
		[]reflect.Value{
			reflect.ValueOf(context.Background()),
			inputV.Addr(),
		},
	)
	callErr := result[1].Interface()
	if callErr != nil {
		panic(callErr)
	}
	response := result[0].Interface()

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", " ")
	err = enc.Encode(response)
	if err != nil {
		panic(fmt.Errorf("unable to serialize the response: %w", err))
	}
	fmt.Printf("%s\n", buf.String())
}

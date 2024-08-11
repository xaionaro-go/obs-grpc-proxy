package main

import (
	"context"
	"log"
	"net"

	"github.com/andreykaipov/goobs"
	"github.com/spf13/pflag"
	"github.com/xaionaro-go/obs-grpc-proxy/pkg/obsgrpcproxy"
	"github.com/xaionaro-go/obs-grpc-proxy/protobuf/go/obs_grpc"
	"google.golang.org/grpc"
)

func main() {
	listenAddr := pflag.String("listen-addr", "localhost:4456", "the address to listen for gRPC connections on")
	obsWSAddr := pflag.String("obs-ws-addr", "localhost:4455", "OBS WebSocket address")
	pflag.Parse()

	listener, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	proxy := obsgrpcproxy.New(func() (*goobs.Client, context.CancelFunc, error) {
		client, err := goobs.New(*obsWSAddr)
		return client, func() { client.Disconnect() }, err
	})

	grpcServer := grpc.NewServer()
	obs_grpc.RegisterOBSServer(grpcServer, proxy)
	err = grpcServer.Serve(listener)
}

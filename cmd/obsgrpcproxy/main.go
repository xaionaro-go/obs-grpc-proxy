package main

import (
	"context"
	"log"
	"net"

	"github.com/andreykaipov/goobs"
	"github.com/andreykaipov/goobs/api/events/subscriptions"
	"github.com/facebookincubator/go-belt/tool/logger"
	xlogrus "github.com/facebookincubator/go-belt/tool/logger/implementation/logrus"
	"github.com/spf13/pflag"
	"github.com/xaionaro-go/obs-grpc-proxy/pkg/obsgrpcproxy"
	"github.com/xaionaro-go/obs-grpc-proxy/protobuf/go/obs_grpc"
	"google.golang.org/grpc"
)

func main() {
	logLevel := logger.LevelInfo
	pflag.Var(&logLevel, "log-level", "Log level")
	listenAddr := pflag.String("listen-addr", "localhost:4456", "the address to listen for gRPC connections on")
	obsWSAddr := pflag.String("obs-ws-addr", "localhost:4455", "OBS WebSocket address")
	obsPassword := pflag.String("obs-password", "", "OBS WebSocket password")
	pflag.Parse()

	ctx := logger.CtxWithLogger(context.Background(), xlogrus.Default().WithLevel(logLevel))

	listener, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	proxy := obsgrpcproxy.New(
		context.Background(),
		func(ctx context.Context) (*goobs.Client, context.CancelFunc, error) {
			client, err := goobs.New(
				*obsWSAddr,
				goobs.WithPassword(*obsPassword),
				goobs.WithEventSubscriptions(subscriptions.All|subscriptions.InputActiveStateChanged),
			)
			logger.Debugf(ctx, "connection to OBS result: %v %v", client, err)
			if err != nil {
				return nil, nil, err
			}
			return client, func() { client.Disconnect() }, err
		},
	)

	grpcServer := grpc.NewServer()
	obs_grpc.RegisterOBSServer(grpcServer, proxy)
	logger.Infof(ctx, "started the server at '%s'", listener.Addr())
	err = grpcServer.Serve(listener)
	logger.Panicf(ctx, "unable to serve gRPC: %v", err)
}

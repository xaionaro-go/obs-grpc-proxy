package commands

import (
	"context"
	"os"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/spf13/cobra"
	"github.com/xaionaro-go/obs-grpc-proxy/pkg/obsdoc"
	"github.com/xaionaro-go/obs-grpc-proxy/pkg/obsprotobufgen"
	"github.com/xaionaro-go/obs-grpc-proxy/pkg/obsproxygen"
)

var (
	Root = &cobra.Command{
		Use: os.Args[0],
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()
			l := logger.FromCtx(ctx).WithLevel(LoggerLevel)
			ctx = logger.CtxWithLogger(ctx, l)
			cmd.SetContext(ctx)
			logger.Debugf(ctx, "log-level: %v", LoggerLevel)
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()
			logger.Debug(ctx, "end")
		},
	}

	Protobuf = &cobra.Command{
		Use:  "protobuf",
		Args: cobra.ExactArgs(2),
		Run:  protobuf,
	}

	Proxy = &cobra.Command{
		Use:  "proxy",
		Args: cobra.ExactArgs(2),
		Run:  proxy,
	}

	LoggerLevel = logger.LevelWarning
)

func init() {
	Root.PersistentFlags().Var(&LoggerLevel, "log-level", "")
	Root.AddCommand(Protobuf)
	Root.AddCommand(Proxy)
}
func assertNoError(ctx context.Context, err error) {
	if err != nil {
		logger.Panic(ctx, err)
	}
}

func protobuf(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()

	protocolFilePath := args[0]
	protobufFilePath := args[1]

	protocolBytes, err := os.ReadFile(protocolFilePath)
	assertNoError(ctx, err)

	protocol, err := obsdoc.ParseProtocol(protocolBytes)
	assertNoError(ctx, err)

	protobufFile, err := os.OpenFile(protobufFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	assertNoError(ctx, err)
	defer protobufFile.Close()

	err = obsprotobufgen.Generate(ctx, protobufFile, protocol)
	assertNoError(ctx, err)
}

func proxy(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()

	protocolFilePath := args[0]
	proxyFilePath := args[1]

	protocolBytes, err := os.ReadFile(protocolFilePath)
	assertNoError(ctx, err)

	protocol, err := obsdoc.ParseProtocol(protocolBytes)
	assertNoError(ctx, err)

	proxyFile, err := os.OpenFile(proxyFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	assertNoError(ctx, err)
	defer proxyFile.Close()

	err = obsproxygen.Generate(ctx, proxyFile, protocol)
	assertNoError(ctx, err)
}

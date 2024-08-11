package commands

import (
	"bytes"
	"context"
	"os"
	"path/filepath"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/spf13/cobra"
	"github.com/xaionaro-go/obs-grpc-proxy/pkg/obsdoc"
	"github.com/xaionaro-go/obs-grpc-proxy/pkg/obsprotobufgen"
	"github.com/xaionaro-go/obs-grpc-proxy/pkg/obsproxygen"
	protoparser "github.com/yoheimuta/go-protoparser/v4"
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
		Args: cobra.ExactArgs(3),
		Run:  protobuf,
	}

	Proxy = &cobra.Command{
		Use:  "proxy",
		Args: cobra.ExactArgs(3),
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
	protobufStaticFilePath := args[1]
	protobufOutFilePath := args[2]

	protocolBytes, err := os.ReadFile(protocolFilePath)
	assertNoError(ctx, err)

	protocol, err := obsdoc.ParseProtocol(protocolBytes)
	assertNoError(ctx, err)

	protobufStaticBytes, err := os.ReadFile(protobufStaticFilePath)
	assertNoError(ctx, err)

	staticProto, err := protoparser.Parse(
		bytes.NewReader(protobufStaticBytes),
		protoparser.WithDebug(false),
		protoparser.WithPermissive(true),
		protoparser.WithFilename(filepath.Base(protobufStaticFilePath)),
	)
	assertNoError(ctx, err)

	protobufFile, err := os.OpenFile(protobufOutFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	assertNoError(ctx, err)
	defer protobufFile.Close()

	err = obsprotobufgen.Generate(ctx, protobufFile, protocol, staticProto)
	assertNoError(ctx, err)
}

func proxy(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()

	protocolFilePath := args[0]
	protobufStaticFilePath := args[1]
	proxyOutFilePath := args[2]

	protocolBytes, err := os.ReadFile(protocolFilePath)
	assertNoError(ctx, err)

	protocol, err := obsdoc.ParseProtocol(protocolBytes)
	assertNoError(ctx, err)

	protobufStaticBytes, err := os.ReadFile(protobufStaticFilePath)
	assertNoError(ctx, err)

	staticProto, err := protoparser.Parse(
		bytes.NewReader(protobufStaticBytes),
		protoparser.WithDebug(false),
		protoparser.WithPermissive(true),
		protoparser.WithFilename(filepath.Base(protobufStaticFilePath)),
	)
	assertNoError(ctx, err)

	proxyFile, err := os.OpenFile(proxyOutFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	assertNoError(ctx, err)
	defer proxyFile.Close()

	err = obsproxygen.Generate(ctx, proxyFile, protocol, staticProto)
	assertNoError(ctx, err)
}

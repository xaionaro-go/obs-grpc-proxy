
all: grpc-go

obs.proto:
	go run ./scripts/generate/ protobuf ./upstream/obs-websocket/docs/generated/protocol.json ./protobuf/obs.proto

grpc-go: obs.proto
	protoc --proto_path=protobuf/ --go_out=protobuf --go-grpc_out=protobuf protobuf/obs.proto

proxy:
	go run ./scripts/generate/ proxy ./upstream/obs-websocket/docs/generated/protocol.json ./pkg/obsgrpcproxy/obsgrpcproxy_gen.go

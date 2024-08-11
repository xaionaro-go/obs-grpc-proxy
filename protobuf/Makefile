
all: grpc-go

obs.proto:
	go run ../scripts/generate/ protobuf ../upstream/obs-websocket/docs/generated/protocol.json ./objects.proto ./obs.proto

grpc-go: obs.proto
	protoc --proto_path=./ --go_out=./ --go-grpc_out=./ ./objects.proto
	protoc --proto_path=./ --go_out=./ --go-grpc_out=./ ./obs.proto

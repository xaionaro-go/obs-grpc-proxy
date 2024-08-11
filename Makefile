
all: proxy

obs.proto:
	make -C protobuf obs.proto

grpc-go: obs.proto
	make -C protobuf grpc-go

proxy: grpc-go
	go run ./scripts/generate/ proxy ./upstream/obs-websocket/docs/generated/protocol.json ./protobuf/objects.proto ./pkg/obsgrpcproxy/obsgrpcproxy_gen.go

// This file was automatically generated by github.com/xaionaro-go/obs-grpc-proxy/scripts/generate

package obsgrpcproxy

import (
	"context"
	"encoding/json"
	"fmt"

	goobs "github.com/andreykaipov/goobs"
	"github.com/xaionaro-go/obs-grpc-proxy/protobuf/go/obs_grpc"
	obsgrpc "github.com/xaionaro-go/obs-grpc-proxy/protobuf/go/obs_grpc"
)

type GetClientFunc func() (*goobs.Client, context.CancelFunc, error)

type Proxy struct {
	obsgrpc.UnimplementedOBSServer

	GetClient GetClientFunc
}

var _ obsgrpc.OBSServer = (*Proxy)(nil)

type ProxyAsClient Proxy

var _ obsgrpc.OBSClient = (*ProxyAsClient)(nil)

type ClientAsServer struct {
	obs_grpc.UnimplementedOBSServer
	obs_grpc.OBSClient
}

var _ obsgrpc.OBSServer = (*ClientAsServer)(nil)

func New(getClient GetClientFunc) *Proxy {
	return &Proxy{
		GetClient: getClient,
	}
}

func ptr[T any](in T) *T {
	return &in
}

func anyGo2Protobuf(in any) *obsgrpc.Any {
	var result obsgrpc.Any
	switch in := in.(type) {
	case []byte:
		result.Union = &obsgrpc.Any_String_{String_: in}
	case string:
		result.Union = &obsgrpc.Any_String_{String_: []byte(in)}
	case int:
		result.Union = &obsgrpc.Any_Integer{Integer: int64(in)}
	case uint:
		result.Union = &obsgrpc.Any_Integer{Integer: int64(in)}
	case int64:
		result.Union = &obsgrpc.Any_Integer{Integer: int64(in)}
	case uint64:
		result.Union = &obsgrpc.Any_Integer{Integer: int64(in)}
	case int32:
		result.Union = &obsgrpc.Any_Integer{Integer: int64(in)}
	case uint32:
		result.Union = &obsgrpc.Any_Integer{Integer: int64(in)}
	case int16:
		result.Union = &obsgrpc.Any_Integer{Integer: int64(in)}
	case uint16:
		result.Union = &obsgrpc.Any_Integer{Integer: int64(in)}
	case int8:
		result.Union = &obsgrpc.Any_Integer{Integer: int64(in)}
	case uint8:
		result.Union = &obsgrpc.Any_Integer{Integer: int64(in)}
	case float64:
		result.Union = &obsgrpc.Any_Float{Float: float64(in)}
	case float32:
		result.Union = &obsgrpc.Any_Float{Float: float64(in)}
	case bool:
		result.Union = &obsgrpc.Any_Bool{Bool: in}
	case map[string]any:
		result.Union = &obsgrpc.Any_Object{
			Object: toAbstractObject(in),
		}
	default:
		panic(fmt.Errorf("unexpected type %T", in))
	}
	return &result
}

func anyProtobuf2Go(in *obsgrpc.Any) any {
	switch in := in.Union.(type) {
	case *obsgrpc.Any_Integer:
		return in.Integer
	case *obsgrpc.Any_Float:
		return in.Float
	case *obsgrpc.Any_String_:
		return string(in.String_)
	case *obsgrpc.Any_Bool:
		return in.Bool
	case *obsgrpc.Any_Object:
		return fromAbstractObject[map[string]any](in.Object)
	default:
		panic(fmt.Errorf("unexpected type: %T", in))
	}
}

func toAbstractObjects[T any](in []T) []*obsgrpc.AbstractObject {
	result := make([]*obsgrpc.AbstractObject, 0, len(in))
	for _, item := range in {
		result = append(result, toAbstractObject(item))
	}
	return result
}

func stringSlice2BytesSlice(in []string) [][]byte {
	var result [][]byte
	for _, s := range in {
		result = append(result, []byte(s))
	}
	return result
}

func ptrInt64ToFloat64(in *int64) *float64 {
	if in == nil {
		return nil
	}

	f := float64(*in)
	return &f
}

func ptrInt64ToInt(in *int64) *int {
	if in == nil {
		return nil
	}

	i := int(*in)
	return &i
}

func toAbstractObject[T any](in T) *obsgrpc.AbstractObject {
	return toAbstractObjectViaJSON(in)
}

func toAbstractObjectViaJSON[T any](in T) *obsgrpc.AbstractObject {
	b, err := json.Marshal(in)
	if err != nil {
		panic(err)
	}
	m := map[string]any{}
	err = json.Unmarshal(b, &m)
	if err != nil {
		panic(err)
	}

	result := &obsgrpc.AbstractObject{
		Fields: map[string]*obsgrpc.Any{},
	}
	for k, v := range m {
		result.Fields[k] = anyGo2Protobuf(v)
	}
	return result
}

func fromAbstractObject[T any](in *obsgrpc.AbstractObject) T {
	return fromAbstractObjectViaJSON[T](in)
}

func fromAbstractObjectViaJSON[T any](in *obsgrpc.AbstractObject) T {
	var result T
	if in == nil || in.Fields == nil {
		return result
	}

	m := map[string]any{}
	for k, f := range in.Fields {
		m[k] = anyProtobuf2Go(f)
	}

	b, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(b, &result)
	if err != nil {
		panic(err)
	}

	return result
}

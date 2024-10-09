package obsgrpcproxy

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	goobs "github.com/andreykaipov/goobs"
	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/xaionaro-go/obs-grpc-proxy/protobuf/go/obs_grpc"
)

type GetClientFunc func(ctx context.Context) (*goobs.Client, context.CancelFunc, error)
type QueryErrorHandler func(ctx context.Context, err error) error

type Proxy struct {
	obs_grpc.UnimplementedOBSServer

	GetClient         GetClientFunc
	QueryErrorHandler QueryErrorHandler
	config            configT
	client            *goobs.Client
	clientCancel      context.CancelFunc
	clientLocker      sync.Mutex
}

var _ obs_grpc.OBSServer = (*Proxy)(nil)

type ProxyAsClient Proxy

var _ obs_grpc.OBSClient = (*ProxyAsClient)(nil)

type ClientAsServer struct {
	obs_grpc.UnimplementedOBSServer
	obs_grpc.OBSClient
}

var _ obs_grpc.OBSServer = (*ClientAsServer)(nil)

func New(
	ctx context.Context,
	getClient GetClientFunc,
	opts ...Option,
) *Proxy {
	proxy := &Proxy{
		GetClient: getClient,
		config:    Options(opts).config(),
	}
	go proxy.processEvents(ctx)
	return proxy
}

func (proxy *Proxy) getClient(
	ctx context.Context,
) (*goobs.Client, error) {
	proxy.clientLocker.Lock()
	defer proxy.clientLocker.Unlock()
	if proxy.client != nil {
		return proxy.client, nil
	}

	var err error
	proxy.client, proxy.clientCancel, err = proxy.GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get a client to OBS: %w", err)
	}
	return proxy.client, nil
}

func (proxy *Proxy) processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		client, err := proxy.getClient(ctx)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
				continue
			}
		}

		func() {
			for {
				select {
				case <-ctx.Done():
					return
				case ev, ok := <-client.IncomingEvents:
					if !ok {
						return
					}
					proxy.processEvent(ctx, ev)
				}
			}
		}()

		func() {
			proxy.clientLocker.Lock()
			defer proxy.clientLocker.Unlock()
			if proxy.clientCancel != nil {
				proxy.clientCancel()
			}
			proxy.client = nil
			proxy.clientCancel = nil
		}()
	}
}

func (proxy *Proxy) processEvent(
	ctx context.Context,
	ev any,
) {
	logger.Tracef(ctx, "received event: %T: %#+v", ev, ev)
	for _, hook := range proxy.config.EventHooks {
		hook.ProcessEvent(ctx, ev)
	}
}

func ptr[T any](in T) *T {
	return &in
}

func AnyGo2Protobuf(in any) *obs_grpc.Any {
	var result obs_grpc.Any
	switch in := in.(type) {
	case []byte:
		result.Union = &obs_grpc.Any_String_{String_: in}
	case string:
		result.Union = &obs_grpc.Any_String_{String_: []byte(in)}
	case int:
		result.Union = &obs_grpc.Any_Integer{Integer: int64(in)}
	case uint:
		result.Union = &obs_grpc.Any_Integer{Integer: int64(in)}
	case int64:
		result.Union = &obs_grpc.Any_Integer{Integer: int64(in)}
	case uint64:
		result.Union = &obs_grpc.Any_Integer{Integer: int64(in)}
	case int32:
		result.Union = &obs_grpc.Any_Integer{Integer: int64(in)}
	case uint32:
		result.Union = &obs_grpc.Any_Integer{Integer: int64(in)}
	case int16:
		result.Union = &obs_grpc.Any_Integer{Integer: int64(in)}
	case uint16:
		result.Union = &obs_grpc.Any_Integer{Integer: int64(in)}
	case int8:
		result.Union = &obs_grpc.Any_Integer{Integer: int64(in)}
	case uint8:
		result.Union = &obs_grpc.Any_Integer{Integer: int64(in)}
	case float64:
		result.Union = &obs_grpc.Any_Float{Float: float64(in)}
	case float32:
		result.Union = &obs_grpc.Any_Float{Float: float64(in)}
	case bool:
		result.Union = &obs_grpc.Any_Bool{Bool: in}
	case map[string]any:
		result.Union = &obs_grpc.Any_Object{
			Object: ToAbstractObject(in),
		}
	default:
		panic(fmt.Errorf("unexpected type %T", in))
	}
	return &result
}

func AnyProtobuf2Go(in *obs_grpc.Any) any {
	switch in := in.Union.(type) {
	case *obs_grpc.Any_Integer:
		return in.Integer
	case *obs_grpc.Any_Float:
		return in.Float
	case *obs_grpc.Any_String_:
		return string(in.String_)
	case *obs_grpc.Any_Bool:
		return in.Bool
	case *obs_grpc.Any_Object:
		result, err := FromAbstractObject[map[string]any](in.Object)
		if err != nil {
			panic(err)
		}
		return result
	default:
		panic(fmt.Errorf("unexpected type: %T", in))
	}
}

func ToAbstractObjects[T any](in []T) []*obs_grpc.AbstractObject {
	result := make([]*obs_grpc.AbstractObject, 0, len(in))
	for _, item := range in {
		result = append(result, ToAbstractObject(item))
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

func ToAbstractObject[T any](in T) *obs_grpc.AbstractObject {
	return toAbstractObjectViaJSON(in)
}

func toAbstractObjectViaJSON[T any](in T) *obs_grpc.AbstractObject {
	b, err := json.Marshal(in)
	if err != nil {
		panic(err)
	}
	m := map[string]any{}
	err = json.Unmarshal(b, &m)
	if err != nil {
		panic(err)
	}

	result := &obs_grpc.AbstractObject{
		Fields: map[string]*obs_grpc.Any{},
	}
	for k, v := range m {
		result.Fields[k] = AnyGo2Protobuf(v)
	}
	return result
}

func FromAbstractObject[T any](in *obs_grpc.AbstractObject) (T, error) {
	return fromAbstractObjectViaJSON[T](in)
}

func fromAbstractObjectViaJSON[T any](in *obs_grpc.AbstractObject) (T, error) {
	var result T
	if in == nil || in.Fields == nil {
		return result, nil
	}

	m := map[string]any{}
	for k, f := range in.Fields {
		m[k] = AnyProtobuf2Go(f)
	}

	b, err := json.Marshal(m)
	if err != nil {
		return result, fmt.Errorf("unable to serialize to JSON: %w", err)
	}

	if reflect.TypeOf(result).Kind() == reflect.Map {
		result = reflect.MakeMap(reflect.TypeOf(result)).Interface().(T)
	}

	err = json.Unmarshal(b, &result)
	if err != nil {
		return result, fmt.Errorf("unable to deserialize from JSON: %w", err)
	}

	return result, nil
}

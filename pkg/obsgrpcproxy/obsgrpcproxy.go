package obsgrpcproxy

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	goobs "github.com/andreykaipov/goobs"
	typedefs "github.com/andreykaipov/goobs/api/typedefs"
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

func FromAbstractObjects[T any](in []*obs_grpc.AbstractObject) ([]T, error) {
	result := make([]T, 0, len(in))
	for idx, item := range in {
		itemConverted, err := FromAbstractObject[T](item)
		if err != nil {
			return nil, fmt.Errorf("unable to convert item #%d (%#+v): %w", idx, item, err)
		}
		result = append(result, itemConverted)
	}
	return result, nil
}

func must[T any](in T, err error) T {
	if err != nil {
		panic(err)
	}
	return in
}

func StreamServiceSettingsGo2Protobuf(
	in *typedefs.StreamServiceSettings,
) *obs_grpc.StreamServiceSettings {
	return must(FromAbstractObject[*obs_grpc.StreamServiceSettings](ToAbstractObject[*typedefs.StreamServiceSettings](in)))
}

func StreamServiceSettingsProtobuf2Go(
	in *obs_grpc.StreamServiceSettings,
) (*typedefs.StreamServiceSettings, error) {
	return FromAbstractObject[*typedefs.StreamServiceSettings](ToAbstractObject[*obs_grpc.StreamServiceSettings](in))
}

func FiltersGo2Protobuf(
	in []*typedefs.Filter,
) []*obs_grpc.Filter {
	return must(FromAbstractObjects[*obs_grpc.Filter](ToAbstractObjects[*typedefs.Filter](in)))
}

func FiltersProtobuf2Go(
	in []*obs_grpc.Filter,
) ([]*typedefs.Filter, error) {
	return FromAbstractObjects[*typedefs.Filter](ToAbstractObjects[*obs_grpc.Filter](in))
}

func KeyModifiersGo2Protobuf(
	in *typedefs.KeyModifiers,
) *obs_grpc.KeyModifiers {
	return must(FromAbstractObject[*obs_grpc.KeyModifiers](ToAbstractObject[*typedefs.KeyModifiers](in)))
}

func KeyModifiersProtobuf2Go(
	in *obs_grpc.KeyModifiers,
) (*typedefs.KeyModifiers, error) {
	return FromAbstractObject[*typedefs.KeyModifiers](ToAbstractObject[*obs_grpc.KeyModifiers](in))
}

func InputsGo2Protobuf(
	in []*typedefs.Input,
) []*obs_grpc.Input {
	return must(FromAbstractObjects[*obs_grpc.Input](ToAbstractObjects[*typedefs.Input](in)))
}

func InputsProtobuf2Go(
	in []*obs_grpc.Input,
) ([]*typedefs.Input, error) {
	return FromAbstractObjects[*typedefs.Input](ToAbstractObjects[*obs_grpc.Input](in))
}

func InputAudioTracksGo2Protobuf(
	in *typedefs.InputAudioTracks,
) *obs_grpc.InputAudioTracks {
	return must(FromAbstractObject[*obs_grpc.InputAudioTracks](ToAbstractObject[*typedefs.InputAudioTracks](in)))
}

func InputAudioTracksProtobuf2Go(
	in *obs_grpc.InputAudioTracks,
) (*typedefs.InputAudioTracks, error) {
	return FromAbstractObject[*typedefs.InputAudioTracks](ToAbstractObject[*obs_grpc.InputAudioTracks](in))
}

func PropertyItemsGo2Protobuf(
	in []*typedefs.PropertyItem,
) []*obs_grpc.PropertyItem {
	return must(FromAbstractObjects[*obs_grpc.PropertyItem](ToAbstractObjects[*typedefs.PropertyItem](in)))
}

func PropertyItemsProtobuf2Go(
	in []*obs_grpc.PropertyItem,
) ([]*typedefs.PropertyItem, error) {
	return FromAbstractObjects[*typedefs.PropertyItem](ToAbstractObjects[*obs_grpc.PropertyItem](in))
}

func OutputsGo2Protobuf(
	in []*typedefs.Output,
) []*obs_grpc.Output {
	return must(FromAbstractObjects[*obs_grpc.Output](ToAbstractObjects[*typedefs.Output](in)))
}

func OutputsProtobuf2Go(
	in []*obs_grpc.Output,
) ([]*typedefs.Output, error) {
	return FromAbstractObjects[*typedefs.Output](ToAbstractObjects[*obs_grpc.Output](in))
}

func SceneItemsGo2Protobuf(
	in []*typedefs.SceneItem,
) []*obs_grpc.SceneItem {
	return must(FromAbstractObjects[*obs_grpc.SceneItem](ToAbstractObjects[*typedefs.SceneItem](in)))
}

func SceneItemsProtobuf2Go(
	in []*obs_grpc.SceneItem,
) ([]*typedefs.SceneItem, error) {
	return FromAbstractObjects[*typedefs.SceneItem](ToAbstractObjects[*obs_grpc.SceneItem](in))
}

func SceneItemTransformGo2Protobuf(
	in *typedefs.SceneItemTransform,
) *obs_grpc.SceneItemTransform {
	return must(FromAbstractObject[*obs_grpc.SceneItemTransform](ToAbstractObject[*typedefs.SceneItemTransform](in)))
}

func SceneItemTransformProtobuf2Go(
	in *obs_grpc.SceneItemTransform,
) (*typedefs.SceneItemTransform, error) {
	return FromAbstractObject[*typedefs.SceneItemTransform](ToAbstractObject[*obs_grpc.SceneItemTransform](in))
}

func ScenesGo2Protobuf(
	in []*typedefs.Scene,
) []*obs_grpc.Scene {
	return must(FromAbstractObjects[*obs_grpc.Scene](ToAbstractObjects[*typedefs.Scene](in)))
}

func ScenesProtobuf2Go(
	in []*obs_grpc.Scene,
) ([]*typedefs.Scene, error) {
	return FromAbstractObjects[*typedefs.Scene](ToAbstractObjects[*obs_grpc.Scene](in))
}

func TransitionsGo2Protobuf(
	in []*typedefs.Transition,
) []*obs_grpc.Transition {
	return must(FromAbstractObjects[*obs_grpc.Transition](ToAbstractObjects[*typedefs.Transition](in)))
}

func TransitionsProtobuf2Go(
	in []*obs_grpc.Transition,
) ([]*typedefs.Transition, error) {
	return FromAbstractObjects[*typedefs.Transition](ToAbstractObjects[*obs_grpc.Transition](in))
}

func MonitorsGo2Protobuf(
	in []*typedefs.Monitor,
) []*obs_grpc.Monitor {
	return must(FromAbstractObjects[*obs_grpc.Monitor](ToAbstractObjects[*typedefs.Monitor](in)))
}

func MonitorsProtobuf2Go(
	in []*obs_grpc.Monitor,
) ([]*typedefs.Monitor, error) {
	return FromAbstractObjects[*typedefs.Monitor](ToAbstractObjects[*obs_grpc.Monitor](in))
}

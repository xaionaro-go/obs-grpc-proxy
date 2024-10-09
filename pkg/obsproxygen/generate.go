package obsproxygen

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/dave/jennifer/jen"
	"github.com/xaionaro-go/obs-grpc-proxy/pkg/obsdoc"
	"github.com/xaionaro-go/obs-grpc-proxy/pkg/obsprotobufgen"
	"github.com/yoheimuta/go-protoparser/v4/parser"
)

func Generate(
	ctx context.Context,
	w io.Writer,
	p *obsdoc.Protocol,
	staticProto *parser.Proto,
) error {
	if p == nil {
		return nil
	}

	existingObjectTypes := map[string]struct{}{}
	if staticProto != nil {
		for _, v := range staticProto.ProtoBody {
			msg, ok := v.(*parser.Message)
			if !ok {
				continue
			}
			existingObjectTypes[msg.MessageName] = struct{}{}
		}
	}

	code := jen.NewFile("obsgrpcproxy")
	code.HeaderComment("This file was automatically generated by github.com/xaionaro-go/obs-grpc-proxy/scripts/generate")
	code.Var().Id("_").Op("=").Params(jen.Id("*").Qual("github.com/andreykaipov/goobs/api/typedefs", "Input")).Call(jen.Nil())

	for idx, request := range p.Requests {
		err := generateRequest(code, request, existingObjectTypes)
		if err != nil {
			return fmt.Errorf("unable to generate code for request #%d:%s: %w", idx, request.RequestType, err)
		}
	}

	err := code.Render(w)
	if err != nil {
		return fmt.Errorf("unable to render the code: %w", err)
	}

	return nil
}

func goOBSIsInt(fieldName string) bool {
	switch {
	case strings.HasSuffix(fieldName, "Index"):
		return true
	case strings.HasSuffix(fieldName, "Id"):
		return true
	}
	return false
}

func goOBSIsFloatNumber(fieldName string) bool {
	switch {
	case strings.HasSuffix(fieldName, "Offset"):
		return true
	case strings.HasSuffix(fieldName, "Cursor"):
		return true
	case strings.HasSuffix(fieldName, "Duration"):
		return true
	case strings.HasSuffix(fieldName, "Position"):
		return true
	case strings.HasSuffix(fieldName, "Millis"):
		return true
	case strings.HasSuffix(fieldName, "Frames"):
		return true
	case strings.HasSuffix(fieldName, "Numerator"):
		return true
	case strings.HasSuffix(fieldName, "Denominator"):
		return true
	case strings.HasSuffix(fieldName, "Width"):
		return true
	case strings.HasSuffix(fieldName, "Height"):
		return true
	case strings.HasSuffix(fieldName, "Mul"):
		return true
	case strings.HasSuffix(fieldName, "Db"):
		return true
	case strings.HasSuffix(fieldName, "Quality"):
		return true
	}
	return false
}

func ptr[T any](in T) *T {
	return &in
}

func generateRequest(
	code *jen.File,
	request obsdoc.Request,
	existingObjectTypes map[string]struct{},
) error {
	var requestFieldPreAssigns []jen.Code
	var requestFieldAssigns []jen.Code
	for _, field := range request.RequestFields {
		if strings.Contains(field.ValueName, ".") {
			continue
		}
		assignField := jen.Id(title(field.ValueName)).Op(":")
		fieldNameSrc := obsprotobufgen.FieldNameObs2Protobuf(title(field.ValueName))
		src := jen.Id("req").Dot(fieldNameSrc)
		castToType := ""
		convertFunc := ""
		convertWithErrFunc := ""
		switch field.ValueType {
		case "String":
			baseTypeFrom := obsprotobufgen.TypeNameObs2Protobuf(field.ValueType, field.ValueName, existingObjectTypes)
			if baseTypeFrom == "bytes" {
				castToType = "string"
				convertFunc = "ptr"
			}
			if !field.ValueOptional {
				convertFunc = "ptr"
			}
		case "Number":
			fieldName := title(field.ValueName)
			if obsprotobufgen.IsFloatNumber(fieldName) {
				if !field.ValueOptional {
					convertFunc = "ptr"
				}
			} else {
				baseTypeTo := "int64"
				baseTypeFrom := "int64"
				switch {
				case goOBSIsInt(fieldName):
					baseTypeTo = "int"
				case goOBSIsFloatNumber(fieldName):
					baseTypeTo = "float64"
				}
				if baseTypeFrom == baseTypeTo {
					if !field.ValueOptional {
						convertFunc = "ptr"
					}
				} else {
					if field.ValueOptional {
						convertFunc = "ptr" + title(baseTypeFrom) + "To" + title(baseTypeTo)
					} else {
						convertFunc = "ptr"
						castToType = baseTypeTo
					}
				}
			}
		case "Boolean":
			if !field.ValueOptional {
				convertFunc = "ptr"
			}
		case "Object":
			fieldName := title(field.ValueName)
			if strings.HasPrefix(field.ValueType, "Array<") {
				fieldName = fieldName[:len(fieldName)-1]
			}
			var typeName string
			if _, ok := existingObjectTypes[fieldName]; ok {
				typeName = jen.Id("*").Qual("github.com/andreykaipov/goobs/api/typedefs", fieldName).GoString()
			} else {
				typeName = "map[string]any"
			}
			convertWithErrFunc = fmt.Sprintf("FromAbstractObject[%s]", typeName)
		}
		if castToType != "" {
			src = jen.Params(jen.Id(castToType)).Params(src)
		}
		if convertWithErrFunc != "" {
			requestFieldPreAssigns = append(
				requestFieldPreAssigns,
				jen.List(jen.Id(untitle(fieldNameSrc)), jen.Id("err")).Op(":=").Add(jen.Id(convertWithErrFunc).Call(src)),
				jen.If(
					jen.Id("err").Op("!=").Nil(),
				).Block(
					jen.Return(
						jen.Nil(),
						jen.Qual("fmt", "Errorf").Call(
							jen.Lit("unable to convert field %s: %w"),
							jen.Lit(fieldNameSrc),
							jen.Id("err"),
						),
					),
				),
			)
			src = jen.Id(untitle(fieldNameSrc))
		}
		if convertFunc != "" {
			src = jen.Id(convertFunc).Call(src)
		}
		assignField = assignField.Add(src).Op(",")
		requestFieldAssigns = append(
			requestFieldAssigns,
			assignField,
		)
	}

	var responseFieldAssigns []jen.Code
	for _, field := range request.ResponseFields {
		assignField := jen.Id(title(obsprotobufgen.FieldNameObs2Protobuf(field.ValueName))).Op(":")
		src := jen.Id("resp").Dot(title(field.ValueName))
		switch field.ValueType {
		case "Any":
			src = jen.Id("AnyGo2Protobuf").Call(src)
		case "Boolean":
		case "String":
			typeName := obsprotobufgen.TypeNameObs2Protobuf(field.ValueType, field.ValueName, existingObjectTypes)
			if typeName == "bytes" {
				src = jen.Params(jen.Id("[]byte")).Call(src)
			}
		case "Number":
			if !obsprotobufgen.IsFloatNumber(field.ValueName) {
				src = jen.Params(jen.Id("int64")).Call(src)
			}
		case "Array<String>":
			typeName := obsprotobufgen.TypeNameObs2Protobuf(field.ValueType, field.ValueName, existingObjectTypes)
			switch typeName {
			case "repeated bytes":
				src = jen.Id("stringSlice2BytesSlice").Call(src)
			}
		default:
			fieldName := title(field.ValueName)
			if strings.HasPrefix(field.ValueType, "Array<") {
				fieldName = fieldName[:len(fieldName)-1]
			}
			var typeName string
			if _, ok := existingObjectTypes[fieldName]; ok {
				typeName = jen.Id("*").Qual("github.com/andreykaipov/goobs/api/typedefs", fieldName).GoString()
			} else {
				typeName = "map[string]any"
			}

			if strings.HasPrefix(field.ValueType, "Array<") {
				src = jen.Id(fmt.Sprintf("ToAbstractObjects[%s]", typeName)).Call(src)
			} else {
				src = jen.Id(fmt.Sprintf("ToAbstractObject[%s]", typeName)).Call(src)
			}
		}
		assignField = assignField.Add(src).Op(",")
		responseFieldAssigns = append(
			responseFieldAssigns,
			assignField,
		)
	}

	var requestFieldAssignCode []jen.Code

	requestFieldAssignCode = append(requestFieldAssignCode, requestFieldPreAssigns...)
	requestFieldAssignCode = append(requestFieldAssignCode, jen.Id("params").Op("=").Op("&").Qual("github.com/andreykaipov/goobs/api/requests/"+categoryObs2GoPkgName(request.Category), request.RequestType+"Params").Block(
		requestFieldAssigns...,
	))

	code.Func().Params(jen.Id("p").Op("*").Id("Proxy")).Id(request.RequestType).Params(
		jen.Id("ctx").Qual("context", "Context"),
		jen.Id("req").Op("*").Qual("github.com/xaionaro-go/obs-grpc-proxy/protobuf/go/obs_grpc", request.RequestType+"Request"),
	).Params(
		jen.Id("_ret").Op("*").Qual("github.com/xaionaro-go/obs-grpc-proxy/protobuf/go/obs_grpc", request.RequestType+"Response"),
		jen.Id("_err").Error(),
	).Block(
		jen.Qual("github.com/facebookincubator/go-belt/tool/logger", "Tracef").Call(jen.Id("ctx"), jen.Lit(request.RequestType)),
		jen.Defer().Func().Params().Block(
			jen.Id("r").Op(":=").Id("recover").Call(),
			jen.If(jen.Id("r").Op("!=").Nil()).Block(
				jen.Id("_err").Op("=").Qual("fmt", "Errorf").Call(jen.Lit("got panic: %v\n\n%s"), jen.Id("r"), jen.Qual("runtime/debug", "Stack").Call()),
			),
			jen.Qual("github.com/facebookincubator/go-belt/tool/logger", "Tracef").Call(jen.Id("ctx"), jen.Lit("/"+request.RequestType+": %v"), jen.Id("_err")),
		).Call(),
		jen.List(jen.Id("client"), jen.Id("err")).Op(":=").Id("p").Dot("getClient").Call(jen.Id("ctx")),
		jen.If(jen.Id("err").Op("!=").Nil()).Block(jen.Return(jen.List(jen.Nil(), jen.Qual("fmt", "Errorf").Params(jen.Lit("unable to get a client: %w"), jen.Id("err"))))),
		jen.Id("params").Op(":=").Op("&").Qual("github.com/andreykaipov/goobs/api/requests/"+categoryObs2GoPkgName(request.Category), request.RequestType+"Params").Block(),
		jen.If(jen.Id("req").Op("!=").Nil()).Block(
			requestFieldAssignCode...,
		),
		jen.Var().Call(
			jen.Id("resp").Op(" ").Op("*").Qual("github.com/andreykaipov/goobs/api/requests/"+categoryObs2GoPkgName(request.Category), request.RequestType+"Response"),
		),
		jen.For().Block(
			jen.List(jen.Id("resp"), jen.Id("err")).Op("=").Id("client").Dot(categoryObs2Go(request.Category)).Dot(request.RequestType).Call(
				jen.Id("params"),
			),
			jen.If(jen.Id("err").Op("!=").Nil().Op("&&").Id("p").Dot("QueryErrorHandler").Op("!=").Nil()).Block(
				jen.Id("fixErr").Op(":=").Id("p").Dot("QueryErrorHandler").Call(jen.Id("ctx"), jen.Id("err")),
				jen.If(jen.Id("fixErr").Op("==").Nil()).Block(
					jen.Qual("github.com/facebookincubator/go-belt/tool/logger", "Tracef").Call(jen.Id("ctx"), jen.Lit("there was error '%s', but it was handled"), jen.Id("err")),
					jen.Continue(),
				),
			),
			jen.Break(),
		),
		jen.If(jen.Id("err").Op("!=").Nil()).Block(jen.Return(jen.List(jen.Nil(), jen.Qual("fmt", "Errorf").Params(jen.Lit("query error: %w"), jen.Id("err"))))),
		jen.If(jen.Id("resp").Op("==").Nil()).Block(jen.Return(jen.List(jen.Nil(), jen.Qual("fmt", "Errorf").Call(jen.Lit("internal error: resp is nil"))))),
		jen.Id("result").Op(":=").Op("&").Qual("github.com/xaionaro-go/obs-grpc-proxy/protobuf/go/obs_grpc", request.RequestType+"Response").Block(responseFieldAssigns...),
		jen.Return(jen.List(jen.Id("result"), jen.Nil())),
	)

	code.Func().Params(jen.Id("p").Op("*").Id("ProxyAsClient")).Id(request.RequestType).Params(
		jen.Id("ctx").Qual("context", "Context"),
		jen.Id("req").Op("*").Qual("github.com/xaionaro-go/obs-grpc-proxy/protobuf/go/obs_grpc", request.RequestType+"Request"),
		jen.Id("opts").Op("...").Qual("google.golang.org/grpc", "CallOption"),
	).Params(
		jen.Op("*").Qual("github.com/xaionaro-go/obs-grpc-proxy/protobuf/go/obs_grpc", request.RequestType+"Response"),
		jen.Error(),
	).Block(
		jen.Return(jen.Params(jen.Id("*Proxy")).Params(jen.Id("p")).Op(".").Id(request.RequestType).Call(
			jen.Id("ctx"),
			jen.Id("req"),
		)),
	)

	code.Func().Params(jen.Id("p").Op("*").Id("ClientAsServer")).Id(request.RequestType).Params(
		jen.Id("ctx").Qual("context", "Context"),
		jen.Id("req").Op("*").Qual("github.com/xaionaro-go/obs-grpc-proxy/protobuf/go/obs_grpc", request.RequestType+"Request"),
	).Params(
		jen.Op("*").Qual("github.com/xaionaro-go/obs-grpc-proxy/protobuf/go/obs_grpc", request.RequestType+"Response"),
		jen.Error(),
	).Block(
		jen.Return(jen.Id("p").Op(".").Id("OBSClient").Op(".").Id(request.RequestType).Call(
			jen.Id("ctx"),
			jen.Id("req"),
		)),
	)

	return nil
}

func title(s string) string {
	if len(s) == 0 {
		return ""
	}

	return strings.ToUpper(s[:1]) + s[1:]
}

func untitle(s string) string {
	if len(s) == 0 {
		return ""
	}

	return strings.ToLower(s[:1]) + s[1:]
}

func categoryObs2Go(obsCat string) string {
	words := strings.Split(obsCat, " ")
	for idx := range words {
		words[idx] = title(words[idx])
	}
	return strings.Join(words, "")
}

func categoryObs2GoPkgName(obsCat string) string {
	words := strings.Split(obsCat, " ")
	for idx := range words {
		words[idx] = strings.ToLower(words[idx])
	}
	return strings.Join(words, "")
}

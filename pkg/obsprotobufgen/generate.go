package obsprotobufgen

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/xaionaro-go/obs-grpc-proxy/pkg/obsdoc"
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

	fmt.Fprintf(w, "syntax = \"proto3\";\n")
	fmt.Fprintf(w, "import public \"objects.proto\";\n")
	fmt.Fprintf(w, "option go_package = \"go/obs_grpc\";\n\n")

	for idx, enum := range p.Enums {
		err := generateEnum(ctx, w, &enum)
		if err != nil {
			return fmt.Errorf("unable to write enum #%d (%s): %#+v: %w", idx, enum.EnumType, enum, err)
		}
	}

	for idx, event := range p.Events {
		err := generateEvent(ctx, w, &event, existingObjectTypes)
		if err != nil {
			return fmt.Errorf("unable to write event #%d (%s): %#+v: %w", idx, event.EventType, event, err)
		}
	}

	err := generateRequests(ctx, w, p.Requests, existingObjectTypes)
	if err != nil {
		return fmt.Errorf("unable to generate requests: %w", err)
	}

	return nil
}

func generateEnum(
	_ context.Context,
	w io.Writer,
	enum *obsdoc.Enum,
) error {
	fmt.Fprintf(w, "enum %s {\n", enum.EnumType)
	for _, value := range enum.EnumIdentifiers {
		if v, ok := value.EnumValue.(int64); !(ok && v == 0) {
			continue
		}
		fmt.Fprintf(w, "\t%s = %v;\n", value.EnumIdentifier, value.EnumValue)
		break
	}
	for _, value := range enum.EnumIdentifiers {
		if v, ok := value.EnumValue.(int64); ok && v == 0 {
			continue
		}
		if value.EnumIdentifier == "None" {
			fmt.Fprintf(w, "\t_%s = %v;\n", value.EnumIdentifier, value.EnumValue)
			continue
		}
		fmt.Fprintf(w, "\t%s = %v;\n", value.EnumIdentifier, value.EnumValue)
	}
	fmt.Fprintf(w, "}\n")
	return nil
}

func generateEvent(
	_ context.Context,
	w io.Writer,
	event *obsdoc.Event,
	existingObjectTypes map[string]struct{},
) error {
	fmt.Fprintf(w, "message Event%s {\n", event.EventType)
	for idx, field := range event.DataFields {
		typeName := TypeNameObs2Protobuf(field.ValueType, field.ValueName, existingObjectTypes)
		fmt.Fprintf(w, "\t%s %v = %d;\n", typeName, FieldNameObs2Protobuf(field.ValueName), idx+1)
	}
	fmt.Fprintf(w, "}\n")
	return nil
}

func title(s string) string {
	if len(s) == 0 {
		return ""
	}

	return strings.ToUpper(s[:1]) + s[1:]
}

var regexpArrayTypeParser = regexp.MustCompile(`Array\<([^>]+)\>`)

func IsFloatNumber(fieldName string) bool {
	switch {
	case strings.HasSuffix(fieldName, "Balance"):
		return true
	}
	return false
}

func TypeNameObs2Protobuf(
	typeName string,
	fieldName string,
	existingObjectTypes map[string]struct{},
) string {
	switch typeName {
	case "String":
		switch {
		case strings.HasSuffix(fieldName, "Id"):
			return "string"
		case strings.HasSuffix(fieldName, "Uuid"):
			return "string"
		case strings.HasSuffix(fieldName, "Name"):
			return "string"
		case strings.HasSuffix(fieldName, "Kind"):
			return "string"
		case strings.HasSuffix(fieldName, "Path"):
			return "string"
		case strings.HasSuffix(fieldName, "Action"):
			return "string"
		default:
			return "bytes"
		}
	case "Boolean":
		return "bool"
	case "Number":
		if IsFloatNumber(fieldName) {
			return "double"
		} else {
			return "int64"
		}
	case "Object":
		if _, ok := existingObjectTypes[fieldName]; ok {
			return title(fieldName)
		}
		return "AbstractObject"
	}

	matches := regexpArrayTypeParser.FindAllStringSubmatch(typeName, -1)
	if matches != nil {
		return "repeated " + TypeNameObs2Protobuf(matches[0][1], fieldName[:len(fieldName)-1], existingObjectTypes)
	}

	return typeName
}

func generateRequests(
	_ context.Context,
	w io.Writer,
	requests []obsdoc.Request,
	existingObjectTypes map[string]struct{},
) error {
	fmt.Fprintf(w, "service OBS {\n")
	for _, request := range requests {
		fmt.Fprintf(w, "\trpc %s(%sRequest) returns (%sResponse) {}\n", request.RequestType, request.RequestType, request.RequestType)
	}
	fmt.Fprintf(w, "}\n")
	for _, request := range requests {
		fmt.Fprintf(w, "message %sRequest {\n", request.RequestType)
		for idx, field := range request.RequestFields {
			fmt.Fprintf(w, "\t%s %v = %d;\n", fieldTypeObs2Protobuf(field, existingObjectTypes), FieldNameObs2Protobuf(field.ValueName), idx+1)
		}
		fmt.Fprintf(w, "}\n")
		fmt.Fprintf(w, "message %sResponse {\n", request.RequestType)
		for idx, field := range request.ResponseFields {
			typeName := TypeNameObs2Protobuf(field.ValueType, field.ValueName, existingObjectTypes)
			fmt.Fprintf(w, "\t%s %v = %d;\n", typeName, FieldNameObs2Protobuf(field.ValueName), idx+1)
		}
		fmt.Fprintf(w, "}\n")
	}

	return nil
}

func FieldNameObs2Protobuf(fieldName string) string {
	fieldName = strings.ReplaceAll(fieldName, ".", "_")
	fieldName = strings.ReplaceAll(fieldName, "Uuid", "UUID")
	if strings.HasSuffix(fieldName, "Id") {
		fieldName = fieldName[:len(fieldName)-2] + "ID"
	}
	return fieldName
}

func fieldTypeObs2Protobuf(
	field obsdoc.Field,
	existingObjectTypes map[string]struct{},
) string {
	typeName := TypeNameObs2Protobuf(field.ValueType, field.ValueName, existingObjectTypes)
	if field.ValueOptional {
		return "optional " + typeName
	}
	return typeName
}

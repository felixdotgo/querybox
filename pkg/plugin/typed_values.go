package plugin

import (
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"
)

// NewValueMap converts general JSON-like values from the app boundary into
// protobuf Value messages for plugin transport.
func NewValueMap(values map[string]interface{}) (map[string]*structpb.Value, error) {
	if len(values) == 0 {
		return nil, nil
	}
	out := make(map[string]*structpb.Value, len(values))
	for key, value := range values {
		pbValue, err := structpb.NewValue(value)
		if err != nil {
			return nil, fmt.Errorf("invalid value for %q: %w", key, err)
		}
		out[key] = pbValue
	}
	return out, nil
}

// SQLArgFromValue converts a protobuf Value into a database/sql argument.
// JSON containers are encoded as compact JSON strings because SQL drivers bind
// scalar values, not arbitrary Go maps/slices.
func SQLArgFromValue(value *structpb.Value) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	switch value.GetKind().(type) {
	case *structpb.Value_NullValue:
		return nil, nil
	case *structpb.Value_NumberValue:
		return value.GetNumberValue(), nil
	case *structpb.Value_StringValue:
		return value.GetStringValue(), nil
	case *structpb.Value_BoolValue:
		return value.GetBoolValue(), nil
	case *structpb.Value_StructValue, *structpb.Value_ListValue:
		encoded, err := json.Marshal(value.AsInterface())
		if err != nil {
			return nil, fmt.Errorf("encode JSON value: %w", err)
		}
		return string(encoded), nil
	default:
		return nil, fmt.Errorf("unsupported protobuf value kind %T", value.GetKind())
	}
}

func IsNullValue(value *structpb.Value) bool {
	if value == nil {
		return true
	}
	_, ok := value.GetKind().(*structpb.Value_NullValue)
	return ok
}

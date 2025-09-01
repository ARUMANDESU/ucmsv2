package otelx

import (
	"fmt"
	"reflect"
	"time"

	"github.com/ARUMANDESU/validation"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func RecordSpanError(span trace.Span, err error, desc string) {
	if span == nil || err == nil {
		return
	}
	if desc == "" {
		desc = err.Error()
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, desc)
}

// SetSpanAttrs sets attributes on a span from a map of key-value pairs.
// It handles various Go types and converts them to appropriate OpenTelemetry attributes.
func SetSpanAttrs(span trace.Span, attrs map[string]any) {
	if span == nil || attrs == nil || len(attrs) == 0 {
		return
	}

	spanAttrs := make([]attribute.KeyValue, 0, len(attrs))

	for key, value := range attrs {
		if attr := convertToAttribute(key, value); attr.Valid() {
			spanAttrs = append(spanAttrs, attr)
		}
	}

	if len(spanAttrs) > 0 {
		span.SetAttributes(spanAttrs...)
	}
}

// convertToAttribute converts a value to an OpenTelemetry attribute.
func convertToAttribute(key string, value any) attribute.KeyValue {
	value, isNil := validation.Indirect(value)
	if isNil {
		return attribute.String(key, "<nil>")
	}

	if attr := handleSpecialTypes(key, value); attr.Valid() {
		return attr
	}

	v := reflect.ValueOf(value)
	return handleReflectValue(key, v)
}

// handleSpecialTypes handles non-reflection based type conversions.
func handleSpecialTypes(key string, value any) attribute.KeyValue {
	switch v := value.(type) {
	case string:
		return attribute.String(key, v)
	case bool:
		return attribute.Bool(key, v)
	case int:
		return attribute.Int(key, v)
	case int64:
		return attribute.Int64(key, v)
	case float64:
		return attribute.Float64(key, v)
	case []string:
		return attribute.StringSlice(key, v)
	case []bool:
		return attribute.BoolSlice(key, v)
	case []int:
		return attribute.IntSlice(key, v)
	case []int64:
		return attribute.Int64Slice(key, v)
	case []float64:
		return attribute.Float64Slice(key, v)
	case []byte:
		return attribute.String(key, string(v))
	case time.Time:
		return attribute.String(key, v.Format(time.RFC3339Nano))
	case *time.Time:
		if v != nil {
			return attribute.String(key, v.Format(time.RFC3339Nano))
		}
		return attribute.String(key, "<nil>")
	case uuid.UUID:
		return attribute.String(key, v.String())
	case fmt.Stringer:
		return attribute.String(key, v.String())
	}

	return attribute.KeyValue{} // Invalid attribute
}

// handleReflectValue handles value conversion using reflection.
func handleReflectValue(key string, v reflect.Value) attribute.KeyValue {
	if !v.IsValid() {
		return attribute.String(key, "<invalid>")
	}

	switch v.Kind() {
	case reflect.String:
		return attribute.String(key, v.String())
	case reflect.Bool:
		return attribute.Bool(key, v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return attribute.Int64(key, v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return attribute.Int64(key, int64(v.Uint()))
	case reflect.Float32, reflect.Float64:
		return attribute.Float64(key, v.Float())
	case reflect.Array:
		return handleArrayType(key, v)
	case reflect.Slice:
		return handleSliceType(key, v)
	case reflect.Struct:
		return handleStructType(key, v)
	case reflect.Ptr:
		if v.IsNil() {
			return attribute.String(key, "<nil>")
		}
		return handleReflectValue(key, v.Elem())
	default:
		if v.CanInterface() {
			if stringer, ok := v.Interface().(fmt.Stringer); ok {
				return attribute.String(key, stringer.String())
			}
		}
		return attribute.String(key, fmt.Sprintf("<unsupported type: %s>", v.Type()))
	}
}

// handleArrayType handles array types, including UUID which is [16]byte.
func handleArrayType(key string, v reflect.Value) attribute.KeyValue {
	if v.Len() == 16 && v.Type().Elem().Kind() == reflect.Uint8 {
		var bytes [16]byte
		for i := range 16 {
			bytes[i] = byte(v.Index(i).Uint())
		}
		if uuidVal, err := uuid.FromBytes(bytes[:]); err == nil {
			return attribute.String(key, uuidVal.String())
		}
	}

	return handleSliceType(key, v)
}

// handleSliceType handles slice types with proper type checking.
func handleSliceType(key string, v reflect.Value) attribute.KeyValue {
	if v.Len() == 0 {
		return attribute.String(key, "[]")
	}

	if v.CanInterface() {
		switch slice := v.Interface().(type) {
		case []string:
			return attribute.StringSlice(key, slice)
		case []bool:
			return attribute.BoolSlice(key, slice)
		case []int:
			return attribute.IntSlice(key, slice)
		case []int64:
			return attribute.Int64Slice(key, slice)
		case []float64:
			return attribute.Float64Slice(key, slice)
		case []byte:
			return attribute.String(key, string(slice))
		}
	}

	// For other slice types, try to build appropriate slice attributes
	elemKind := v.Type().Elem().Kind()
	switch elemKind {
	case reflect.String:
		return attribute.StringSlice(key, v.Interface().([]string))
	case reflect.Bool:
		return attribute.BoolSlice(key, v.Interface().([]bool))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		ints := make([]int64, v.Len())
		for i := 0; i < v.Len(); i++ {
			ints[i] = v.Index(i).Int()
		}
		return attribute.Int64Slice(key, ints)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		ints := make([]int64, v.Len())
		for i := 0; i < v.Len(); i++ {
			ints[i] = int64(v.Index(i).Uint())
		}
		return attribute.Int64Slice(key, ints)
	case reflect.Float32, reflect.Float64:
		floats := make([]float64, v.Len())
		for i := 0; i < v.Len(); i++ {
			floats[i] = v.Index(i).Float()
		}
		return attribute.Float64Slice(key, floats)
	default:
		// For complex types, convert to string representation
		strs := make([]string, v.Len())
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			if elem.CanInterface() {
				if stringer, ok := elem.Interface().(fmt.Stringer); ok {
					strs[i] = stringer.String()
				} else {
					strs[i] = fmt.Sprintf("%v", elem.Interface())
				}
			} else {
				strs[i] = "<unexported>"
			}
		}
		return attribute.StringSlice(key, strs)
	}
}

// handleStructType handles struct types, checking for special cases like time.Time.
func handleStructType(key string, v reflect.Value) attribute.KeyValue {
	if !v.CanInterface() {
		return attribute.String(key, "<unexported struct>")
	}

	if t, ok := v.Interface().(time.Time); ok {
		return attribute.String(key, t.Format(time.RFC3339Nano))
	}

	if stringer, ok := v.Interface().(fmt.Stringer); ok {
		return attribute.String(key, stringer.String())
	}

	return attribute.String(key, fmt.Sprintf("%+v", v.Interface()))
}

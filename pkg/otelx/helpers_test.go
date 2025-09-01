package otelx

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

type testStringer struct{ val string }

func (ts testStringer) String() string {
	return ts.val
}

type UUID uuid.UUID

type String string

func TestSetSpanAttrs(t *testing.T) {
	t.Run("Nil span", func(t *testing.T) {
		SetSpanAttrs(nil, map[string]any{"key": "value"})
	})

	t.Run("Nil attrs", func(t *testing.T) {
		exporter := tracetest.NewInMemoryExporter()
		provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
		tracer := provider.Tracer("test")
		_, span := tracer.Start(context.TODO(), "test")

		SetSpanAttrs(span, nil)
		span.End()

		spans := exporter.GetSpans()
		assert.Len(t, spans, 1)
		assert.Empty(t, spans[0].Attributes)
	})

	t.Run("Empty attrs", func(t *testing.T) {
		exporter := tracetest.NewInMemoryExporter()
		provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
		tracer := provider.Tracer("test")
		_, span := tracer.Start(context.TODO(), "test")

		SetSpanAttrs(span, map[string]any{})
		span.End()

		spans := exporter.GetSpans()
		assert.Len(t, spans, 1)
		assert.Empty(t, spans[0].Attributes)
	})

	t.Run("Basic types", func(t *testing.T) {
		exporter := tracetest.NewInMemoryExporter()
		provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
		tracer := provider.Tracer("test")
		_, span := tracer.Start(context.TODO(), "test")

		attrs := map[string]any{
			"str":       "test",
			"customStr": String("custom"),
			"bool":      true,
			"int":       42,
			"int64":     int64(1234),
			"float64":   3.14,
		}

		SetSpanAttrs(span, attrs)
		span.End()

		spans := exporter.GetSpans()
		assert.Len(t, spans, 1)

		expectedAttrs := []attribute.KeyValue{
			attribute.String("str", "test"),
			attribute.String("customStr", "custom"),
			attribute.Bool("bool", true),
			attribute.Int("int", 42),
			attribute.Int64("int64", 1234),
			attribute.Float64("float64", 3.14),
		}

		for _, expected := range expectedAttrs {
			assert.Contains(t, spans[0].Attributes, expected)
		}
	})

	t.Run("Slice types", func(t *testing.T) {
		exporter := tracetest.NewInMemoryExporter()
		provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
		tracer := provider.Tracer("test")
		_, span := tracer.Start(context.TODO(), "test")

		attrs := map[string]any{
			"bytes":   []byte("test"),
			"strings": []string{"a", "b"},
			"bools":   []bool{true, false},
			"ints":    []int{1, 2},
			"int64s":  []int64{1, 2},
			"floats":  []float64{1.1, 2.2},
		}

		SetSpanAttrs(span, attrs)
		span.End()

		spans := exporter.GetSpans()
		assert.Len(t, spans, 1)

		expectedAttrs := []attribute.KeyValue{
			attribute.String("bytes", "test"),
			attribute.StringSlice("strings", []string{"a", "b"}),
			attribute.BoolSlice("bools", []bool{true, false}),
			attribute.IntSlice("ints", []int{1, 2}),
			attribute.Int64Slice("int64s", []int64{1, 2}),
			attribute.Float64Slice("floats", []float64{1.1, 2.2}),
		}

		for _, expected := range expectedAttrs {
			assert.Contains(t, spans[0].Attributes, expected)
		}
	})

	t.Run("Time types", func(t *testing.T) {
		exporter := tracetest.NewInMemoryExporter()
		provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
		tracer := provider.Tracer("test")
		_, span := tracer.Start(context.TODO(), "test")

		timeVal := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
		attrs := map[string]any{
			"time":    timeVal,
			"timePtr": &timeVal,
			"nilTime": (*time.Time)(nil),
		}

		SetSpanAttrs(span, attrs)
		span.End()

		spans := exporter.GetSpans()
		assert.Len(t, spans, 1)

		assert.Contains(t, spans[0].Attributes, attribute.String("time", "2023-01-01T12:00:00Z"))
		assert.Contains(t, spans[0].Attributes, attribute.String("timePtr", "2023-01-01T12:00:00Z"))
	})

	t.Run("UUID types", func(t *testing.T) {
		exporter := tracetest.NewInMemoryExporter()
		provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
		tracer := provider.Tracer("test")
		_, span := tracer.Start(context.TODO(), "test")

		uuidVal := uuid.New()
		attrs := map[string]any{
			"uuid":    uuidVal,
			"uuidPtr": &uuidVal,
			"nilUUID": (*uuid.UUID)(nil),
			"custom":  UUID(uuidVal),
		}

		SetSpanAttrs(span, attrs)
		span.End()

		spans := exporter.GetSpans()
		assert.Len(t, spans, 1)

		assert.Contains(t, spans[0].Attributes, attribute.String("uuid", uuidVal.String()))
		assert.Contains(t, spans[0].Attributes, attribute.String("uuidPtr", uuidVal.String()))
		assert.Contains(t, spans[0].Attributes, attribute.String("custom", uuidVal.String()))
	})

	t.Run("Stringer interface", func(t *testing.T) {
		exporter := tracetest.NewInMemoryExporter()
		provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
		tracer := provider.Tracer("test")
		_, span := tracer.Start(context.TODO(), "test")

		attrs := map[string]any{"stringer": testStringer{val: "custom"}}

		SetSpanAttrs(span, attrs)
		span.End()

		spans := exporter.GetSpans()
		assert.Len(t, spans, 1)
		assert.Contains(t, spans[0].Attributes, attribute.String("stringer", "custom"))
	})

	t.Run("Unsupported type", func(t *testing.T) {
		exporter := tracetest.NewInMemoryExporter()
		provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
		tracer := provider.Tracer("test")
		_, span := tracer.Start(context.TODO(), "test")

		attrs := map[string]any{
			"supported":   "value",
			"unsupported": make(chan int),
		}

		SetSpanAttrs(span, attrs)
		span.End()

		spans := exporter.GetSpans()
		assert.Len(t, spans, 1)
		assert.Contains(t, spans[0].Attributes, attribute.String("supported", "value"))
		assert.Contains(t, spans[0].Attributes, attribute.String("unsupported", "<unsupported type: chan int>"))
		assert.Len(t, spans[0].Attributes, 2)
	})
}

func BenchmarkSetSpanAttrs_SmallMap(b *testing.B) {
	exporter := tracetest.NewInMemoryExporter()
	provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
	tracer := provider.Tracer("test")

	attrs := map[string]any{
		"str":  "test",
		"int":  42,
		"bool": true,
	}

	for b.Loop() {
		_, span := tracer.Start(context.TODO(), "test")
		SetSpanAttrs(span, attrs)
		span.End()
	}
}

func BenchmarkSetSpanAttrs_MediumMap(b *testing.B) {
	exporter := tracetest.NewInMemoryExporter()
	provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
	tracer := provider.Tracer("test")

	attrs := map[string]any{
		"str":       "test",
		"customStr": String("custom"),
		"bool":      true,
		"int":       42,
		"int64":     int64(1234),
		"float64":   3.14,
		"bytes":     []byte("test"),
		"strings":   []string{"a", "b"},
		"bools":     []bool{true, false},

		"ints":   []int{1, 2},
		"int64s": []int64{1, 2},
		"floats": []float64{1.1, 2.2},
	}

	for b.Loop() {
		_, span := tracer.Start(context.TODO(), "test")
		SetSpanAttrs(span, attrs)
		span.End()
	}
}

func BenchmarkSetSpanAttrs_LargeMap(b *testing.B) {
	exporter := tracetest.NewInMemoryExporter()
	provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
	tracer := provider.Tracer("test")

	attrs := make(map[string]any, 50)
	for i := range 50 {
		switch i % 7 {
		case 0:
			attrs[fmt.Sprintf("str_%d", i)] = fmt.Sprintf("value_%d", i)
		case 1:
			attrs[fmt.Sprintf("int_%d", i)] = i
		case 2:
			attrs[fmt.Sprintf("bool_%d", i)] = i%2 == 0
		case 3:
			attrs[fmt.Sprintf("float_%d", i)] = float64(i) * 1.1
		case 4:
			attrs[fmt.Sprintf("int64_%d", i)] = int64(i * 1000)
		case 5:
			attrs[fmt.Sprintf("bytes_%d", i)] = fmt.Appendf(nil, "bytes_%d", i)
		case 6:
			attrs[fmt.Sprintf("strings_%d", i)] = []string{fmt.Sprintf("a_%d", i), fmt.Sprintf("b_%d", i)}
		}
	}

	for b.Loop() {
		_, span := tracer.Start(context.TODO(), "test")
		SetSpanAttrs(span, attrs)
		span.End()
	}
}

func BenchmarkSetSpanAttrs_StringTypes(b *testing.B) {
	exporter := tracetest.NewInMemoryExporter()
	provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
	tracer := provider.Tracer("test")

	attrs := map[string]any{
		"short_str":   "test",
		"medium_str":  strings.Repeat("medium", 10),
		"long_str":    strings.Repeat("long string content", 100),
		"custom_str":  String("custom type"),
		"empty_str":   "",
		"unicode_str": "测试🌍",
	}

	for b.Loop() {
		_, span := tracer.Start(context.TODO(), "test")
		SetSpanAttrs(span, attrs)
		span.End()
	}
}

func BenchmarkSetSpanAttrs_NumericTypes(b *testing.B) {
	exporter := tracetest.NewInMemoryExporter()
	provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
	tracer := provider.Tracer("test")

	attrs := map[string]any{
		"int":        42,
		"int8":       int8(127),
		"int16":      int16(32767),
		"int32":      int32(2147483647),
		"int64":      int64(9223372036854775807),
		"uint":       uint(42),
		"uint8":      uint8(255),
		"uint16":     uint16(65535),
		"uint32":     uint32(4294967295),
		"uint64":     uint64(18446744073709551615),
		"float32":    float32(3.14),
		"float64":    float64(3.14159265359),
		"zero_int":   0,
		"neg_int":    -42,
		"zero_float": 0.0,
		"neg_float":  -3.14,
	}

	for b.Loop() {
		_, span := tracer.Start(context.TODO(), "test")
		SetSpanAttrs(span, attrs)
		span.End()
	}
}

func BenchmarkSetSpanAttrs_ArrayTypes(b *testing.B) {
	exporter := tracetest.NewInMemoryExporter()
	provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
	tracer := provider.Tracer("test")

	attrs := map[string]any{
		"empty_strings": []string{},
		"short_strings": []string{"a", "b", "c"},
		"long_strings":  []string{strings.Repeat("long", 50), strings.Repeat("string", 50)},
		"empty_ints":    []int{},
		"short_ints":    []int{1, 2, 3, 4, 5},
		"long_ints":     make([]int, 100),
		"empty_bools":   []bool{},
		"short_bools":   []bool{true, false, true},
		"long_bools":    make([]bool, 100),
		"empty_floats":  []float64{},
		"short_floats":  []float64{1.1, 2.2, 3.3},
		"long_floats":   make([]float64, 100),
		"bytes":         []byte("test bytes content"),
		"large_bytes":   make([]byte, 1000),
	}

	// Fill long arrays
	longInts := attrs["long_ints"].([]int)
	for i := range longInts {
		longInts[i] = i
	}
	longBools := attrs["long_bools"].([]bool)
	for i := range longBools {
		longBools[i] = i%2 == 0
	}
	longFloats := attrs["long_floats"].([]float64)
	for i := range longFloats {
		longFloats[i] = float64(i) * 1.1
	}
	largeBytes := attrs["large_bytes"].([]byte)
	for i := range largeBytes {
		largeBytes[i] = byte(i % 256)
	}

	for b.Loop() {
		_, span := tracer.Start(context.TODO(), "test")
		SetSpanAttrs(span, attrs)
		span.End()
	}
}

func BenchmarkSetSpanAttrs_EdgeCases(b *testing.B) {
	exporter := tracetest.NewInMemoryExporter()
	provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
	tracer := provider.Tracer("test")

	attrs := map[string]any{
		"nil_value":     nil,
		"zero_int":      0,
		"zero_float":    0.0,
		"false_bool":    false,
		"empty_string":  "",
		"empty_bytes":   []byte{},
		"empty_strings": []string{},
		"empty_ints":    []int{},
		"empty_bools":   []bool{},
		"empty_floats":  []float64{},
		"whitespace":    "   ",
		"newlines":      "\n\t\r",
	}

	for b.Loop() {
		_, span := tracer.Start(context.TODO(), "test")
		SetSpanAttrs(span, attrs)
		span.End()
	}
}

func BenchmarkSetSpanAttrs_EmptyMap(b *testing.B) {
	exporter := tracetest.NewInMemoryExporter()
	provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
	tracer := provider.Tracer("test")

	attrs := map[string]any{}

	for b.Loop() {
		_, span := tracer.Start(context.TODO(), "test")
		SetSpanAttrs(span, attrs)
		span.End()
	}
}

func BenchmarkSetSpanAttrs_NilMap(b *testing.B) {
	exporter := tracetest.NewInMemoryExporter()
	provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
	tracer := provider.Tracer("test")

	var attrs map[string]any

	for b.Loop() {
		_, span := tracer.Start(context.TODO(), "test")
		SetSpanAttrs(span, attrs)
		span.End()
	}
}

func BenchmarkSetSpanAttrs_MixedComplexTypes(b *testing.B) {
	exporter := tracetest.NewInMemoryExporter()
	provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
	tracer := provider.Tracer("test")

	attrs := map[string]any{
		"user.id":          "user_12345",
		"user.email":       "user@example.com",
		"user.permissions": []string{"read", "write", "admin"},
		"request.method":   "POST",
		"request.path":     "/api/v1/users",
		"request.headers":  []string{"Content-Type: application/json", "Authorization: Bearer token"},
		"response.status":  200,
		"response.size":    int64(1024),
		"timing.start":     int64(1640995200000),
		"timing.duration":  float64(125.5),
		"flags.enabled":    []bool{true, false, true, true},
		"metadata.tags":    []string{"critical", "user-facing", "api"},
		"error.occurred":   false,
		"cache.hit":        true,
		"db.queries":       []int{1, 2, 5, 3},
		"custom.data":      String("custom_value"),
	}

	for b.Loop() {
		_, span := tracer.Start(context.TODO(), "test")
		SetSpanAttrs(span, attrs)
		span.End()
	}
}

func BenchmarkSetSpanAttrs_RealisticHTTPRequest(b *testing.B) {
	exporter := tracetest.NewInMemoryExporter()
	provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
	tracer := provider.Tracer("test")

	attrs := map[string]any{
		"http.method":          "POST",
		"http.url":             "https://api.example.com/v1/users/12345/profile",
		"http.status_code":     200,
		"http.user_agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"http.request_size":    int64(256),
		"http.response_size":   int64(1024),
		"user.id":              "12345",
		"user.role":            "admin",
		"request.trace_id":     "abc123def456",
		"request.span_id":      "span789",
		"db.connection_string": "postgresql://localhost:5432/mydb",
		"db.statement":         "SELECT * FROM users WHERE id = $1",
		"db.rows_affected":     int64(1),
		"cache.keys":           []string{"user:12345", "profile:12345"},
		"cache.hit_ratio":      0.85,
		"service.name":         "user-service",
		"service.version":      "v1.2.3",
		"deployment.env":       "production",
		"error":                false,
		"latency_ms":           125.5,
	}

	for b.Loop() {
		_, span := tracer.Start(context.TODO(), "test")
		SetSpanAttrs(span, attrs)
		span.End()
	}
}

// Benchmark comparison with direct attribute setting
func BenchmarkDirectSpanAttrs_Comparison(b *testing.B) {
	exporter := tracetest.NewInMemoryExporter()
	provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
	tracer := provider.Tracer("test")

	for b.Loop() {
		_, span := tracer.Start(context.TODO(), "test")
		span.SetAttributes(
			attribute.String("str", "test"),
			attribute.String("customStr", string(String("custom"))),
			attribute.Bool("bool", true),
			attribute.Int("int", 42),
			attribute.Int64("int64", int64(1234)),
			attribute.Float64("float64", 3.14),
			// Note: Direct API doesn't support all types like []byte, []string, etc.
		)
		span.End()
	}
}

// Table-driven benchmark for different scenarios
func BenchmarkSetSpanAttrs_TableDriven(b *testing.B) {
	testCases := []struct {
		name  string
		attrs map[string]any
	}{
		{
			name: "SingleString",
			attrs: map[string]any{
				"key": "value",
			},
		},
		{
			name: "MultipleBasicTypes",
			attrs: map[string]any{
				"str":   "test",
				"int":   42,
				"bool":  true,
				"float": 3.14,
			},
		},
		{
			name: "Arrays",
			attrs: map[string]any{
				"strings": []string{"a", "b", "c"},
				"ints":    []int{1, 2, 3},
				"bools":   []bool{true, false, true},
			},
		},
		{
			name: "LargeString",
			attrs: map[string]any{
				"large": strings.Repeat("x", 1000),
			},
		},
		{
			name: "LargeArray",
			attrs: map[string]any{
				"large_array": make([]string, 100),
			},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			exporter := tracetest.NewInMemoryExporter()
			provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
			tracer := provider.Tracer("test")

			// Initialize large array if needed
			if largeArray, ok := tc.attrs["large_array"].([]string); ok {
				for i := range largeArray {
					largeArray[i] = fmt.Sprintf("item_%d", i)
				}
			}

			for b.Loop() {
				_, span := tracer.Start(context.TODO(), "test")
				SetSpanAttrs(span, tc.attrs)
				span.End()
			}
		})
	}
}

// Memory allocation benchmark
func BenchmarkSetSpanAttrs_Memory(b *testing.B) {
	exporter := tracetest.NewInMemoryExporter()
	provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
	tracer := provider.Tracer("test")

	attrs := map[string]any{
		"str":     "test",
		"int":     42,
		"bool":    true,
		"float":   3.14,
		"strings": []string{"a", "b", "c"},
		"ints":    []int{1, 2, 3},
	}

	b.ReportAllocs()

	for b.Loop() {
		_, span := tracer.Start(context.TODO(), "test")
		SetSpanAttrs(span, attrs)
		span.End()
	}
}

// Benchmark for worst-case scenario with many different types
func BenchmarkSetSpanAttrs_WorstCase(b *testing.B) {
	exporter := tracetest.NewInMemoryExporter()
	provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
	tracer := provider.Tracer("test")

	// Create a large map with all supported types mixed together
	attrs := make(map[string]any, 100)

	// Add many different string values
	for i := range 20 {
		attrs[fmt.Sprintf("str_%d", i)] = strings.Repeat(fmt.Sprintf("value_%d_", i), 10)
	}

	// Add many arrays of different types and sizes
	for i := range 20 {
		attrs[fmt.Sprintf("strings_%d", i)] = make([]string, i+1)
		attrs[fmt.Sprintf("ints_%d", i)] = make([]int, i+1)
		attrs[fmt.Sprintf("bools_%d", i)] = make([]bool, i+1)
		attrs[fmt.Sprintf("floats_%d", i)] = make([]float64, i+1)
	}

	// Add various numeric types
	for i := range 10 {
		attrs[fmt.Sprintf("int_%d", i)] = i
		attrs[fmt.Sprintf("int64_%d", i)] = int64(i * 1000)
		attrs[fmt.Sprintf("float_%d", i)] = float64(i) * 1.1
		attrs[fmt.Sprintf("bool_%d", i)] = i%2 == 0
	}

	// Add large byte arrays
	for i := range 10 {
		attrs[fmt.Sprintf("bytes_%d", i)] = make([]byte, (i+1)*100)
	}

	// Add custom types
	for i := range 10 {
		attrs[fmt.Sprintf("custom_%d", i)] = String(fmt.Sprintf("custom_%d", i))
	}

	// Initialize all the arrays with data
	for key, value := range attrs {
		switch v := value.(type) {
		case []string:
			for i := range v {
				v[i] = fmt.Sprintf("%s_item_%d", key, i)
			}
		case []int:
			for i := range v {
				v[i] = i
			}
		case []bool:
			for i := range v {
				v[i] = i%2 == 0
			}
		case []float64:
			for i := range v {
				v[i] = float64(i) * 1.1
			}
		case []byte:
			for i := range v {
				v[i] = byte(i % 256)
			}
		}
	}

	b.ReportAllocs()

	for b.Loop() {
		_, span := tracer.Start(context.TODO(), "test")
		SetSpanAttrs(span, attrs)
		span.End()
	}
}

// Benchmark for concurrent usage
func BenchmarkSetSpanAttrs_Concurrent(b *testing.B) {
	exporter := tracetest.NewInMemoryExporter()
	provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
	tracer := provider.Tracer("test")

	attrs := map[string]any{
		"str":     "test",
		"int":     42,
		"bool":    true,
		"strings": []string{"a", "b", "c"},
		"ints":    []int{1, 2, 3},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, span := tracer.Start(context.TODO(), "test")
			SetSpanAttrs(span, attrs)
			span.End()
		}
	})
}

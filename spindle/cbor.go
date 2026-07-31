package spindle

import (
	"bytes"
	"fmt"
	"io"

	cborgen "github.com/whyrusleeping/cbor-gen"
)

// decodeCBORMaps decodes consecutive CBOR maps from a WebSocket binary frame.
// Each spindle log frame is a header map followed by a body map.
func decodeCBORMaps(data []byte) ([]map[string]any, error) {
	r := bytes.NewReader(data)
	var maps []map[string]any
	for r.Len() > 0 {
		m, err := decodeCBORMap(r)
		if err != nil {
			return nil, err
		}
		maps = append(maps, m)
	}
	return maps, nil
}

func decodeCBORMap(r io.Reader) (map[string]any, error) {
	majorType, length, err := cborgen.CborReadHeader(r)
	if err != nil {
		return nil, fmt.Errorf("read map header: %w", err)
	}
	if majorType != 5 {
		return nil, fmt.Errorf("expected CBOR map, got major type %d", majorType)
	}
	result := make(map[string]any, length)
	for i := uint64(0); i < length; i++ {
		key, err := decodeCBORValue(r)
		if err != nil {
			return nil, fmt.Errorf("read map key %d: %w", i, err)
		}
		strKey, ok := key.(string)
		if !ok {
			return nil, fmt.Errorf("expected string map key, got %T", key)
		}
		value, err := decodeCBORValue(r)
		if err != nil {
			return nil, fmt.Errorf("read map value for %q: %w", strKey, err)
		}
		result[strKey] = value
	}
	return result, nil
}

func decodeCBORValue(r io.Reader) (any, error) {
	majorType, value, err := cborgen.CborReadHeader(r)
	if err != nil {
		return nil, err
	}
	switch majorType {
	case 0:
		return value, nil
	case 1:
		return -int64(value) - 1, nil
	case 3:
		buf := make([]byte, value)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, fmt.Errorf("read string: %w", err)
		}
		return string(buf), nil
	case 7:
		switch value {
		case 20:
			return false, nil
		case 21:
			return true, nil
		case 22:
			return nil, nil
		default:
			return nil, fmt.Errorf("unsupported simple value %d", value)
		}
	default:
		return nil, fmt.Errorf("unsupported CBOR major type %d", majorType)
	}
}

func mapString(m map[string]any, key string) string {
	if s, ok := m[key].(string); ok {
		return s
	}
	return ""
}

func mapInt(m map[string]any, key string) int {
	switch n := m[key].(type) {
	case int64:
		return int(n)
	case int:
		return n
	}
	return 0
}

func mapStringOpt(m map[string]any, key string) *string {
	if s, ok := m[key].(string); ok && s != "" {
		return &s
	}
	return nil
}

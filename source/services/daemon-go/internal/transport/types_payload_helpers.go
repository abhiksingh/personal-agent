package transport

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

func setStringField(target map[string]any, key string, value string) {
	if strings.TrimSpace(key) == "" {
		return
	}
	if strings.TrimSpace(value) != "" {
		target[key] = strings.TrimSpace(value)
	}
}

func setRawStringField(target map[string]any, key string, value string) {
	if strings.TrimSpace(key) == "" {
		return
	}
	if value != "" {
		target[key] = value
	}
}

func setStringSliceField(target map[string]any, key string, values []string) {
	if strings.TrimSpace(key) == "" || len(values) == 0 {
		return
	}
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		cleaned = append(cleaned, trimmed)
	}
	if len(cleaned) == 0 {
		return
	}
	target[key] = cleaned
}

func setIntPointerField(target map[string]any, key string, value *int) {
	if strings.TrimSpace(key) == "" || value == nil {
		return
	}
	target[key] = *value
}

func setBoolPointerField(target map[string]any, key string, value *bool) {
	if strings.TrimSpace(key) == "" || value == nil {
		return
	}
	target[key] = *value
}

func setAnyMapField(target map[string]any, key string, value map[string]any) {
	if strings.TrimSpace(key) == "" || len(value) == 0 {
		return
	}
	target[key] = cloneAnyMapShallow(value)
}

func cloneAnyMapShallow(value map[string]any) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	result := make(map[string]any, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}

func removeKnownKeys(value map[string]any, knownKeys ...string) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	known := map[string]struct{}{}
	for _, key := range knownKeys {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		known[trimmed] = struct{}{}
	}
	result := map[string]any{}
	for key, item := range value {
		if _, exists := known[strings.TrimSpace(key)]; exists {
			continue
		}
		result[key] = item
	}
	return result
}

func readAnyString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	case json.Number:
		return strings.TrimSpace(typed.String())
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case int32:
		return strconv.FormatInt(int64(typed), 10)
	case int16:
		return strconv.FormatInt(int64(typed), 10)
	case int8:
		return strconv.FormatInt(int64(typed), 10)
	case uint:
		return strconv.FormatUint(uint64(typed), 10)
	case uint64:
		return strconv.FormatUint(typed, 10)
	case uint32:
		return strconv.FormatUint(uint64(typed), 10)
	case uint16:
		return strconv.FormatUint(uint64(typed), 10)
	case uint8:
		return strconv.FormatUint(uint64(typed), 10)
	case float64:
		return strings.TrimSpace(strconv.FormatFloat(typed, 'f', -1, 64))
	case float32:
		return strings.TrimSpace(strconv.FormatFloat(float64(typed), 'f', -1, 32))
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

func readAnyStringPreservingWhitespace(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case []byte:
		return string(typed)
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func readAnyBoolPointer(value any) *bool {
	switch typed := value.(type) {
	case nil:
		return nil
	case bool:
		result := typed
		return &result
	case string:
		trimmed := strings.ToLower(strings.TrimSpace(typed))
		if trimmed == "" {
			return nil
		}
		if trimmed == "true" || trimmed == "1" || trimmed == "yes" {
			result := true
			return &result
		}
		if trimmed == "false" || trimmed == "0" || trimmed == "no" {
			result := false
			return &result
		}
		return nil
	case json.Number:
		if intVal, err := typed.Int64(); err == nil {
			result := intVal != 0
			return &result
		}
		if floatVal, err := typed.Float64(); err == nil {
			result := floatVal != 0
			return &result
		}
		return nil
	case int:
		result := typed != 0
		return &result
	case int64:
		result := typed != 0
		return &result
	case float64:
		result := typed != 0
		return &result
	default:
		return nil
	}
}

func readAnyIntPointer(value any) *int {
	switch typed := value.(type) {
	case nil:
		return nil
	case int:
		result := typed
		return &result
	case int64:
		if converted, ok := intFromInt64Checked(typed); ok {
			return &converted
		}
		return nil
	case int32:
		result := int(typed)
		return &result
	case int16:
		result := int(typed)
		return &result
	case int8:
		result := int(typed)
		return &result
	case uint:
		if converted, ok := intFromUint64Checked(uint64(typed)); ok {
			return &converted
		}
		return nil
	case uint64:
		if converted, ok := intFromUint64Checked(typed); ok {
			return &converted
		}
		return nil
	case uint32:
		if converted, ok := intFromUint64Checked(uint64(typed)); ok {
			return &converted
		}
		return nil
	case uint16:
		result := int(typed)
		return &result
	case uint8:
		result := int(typed)
		return &result
	case float64:
		if converted, ok := intFromFloat64Checked(typed); ok {
			return &converted
		}
		return nil
	case float32:
		if converted, ok := intFromFloat64Checked(float64(typed)); ok {
			return &converted
		}
		return nil
	case json.Number:
		trimmed := strings.TrimSpace(typed.String())
		if trimmed == "" {
			return nil
		}
		if intVal, err := typed.Int64(); err == nil {
			if converted, ok := intFromInt64Checked(intVal); ok {
				return &converted
			}
			return nil
		}
		if uintVal, err := strconv.ParseUint(trimmed, 10, 64); err == nil {
			if converted, ok := intFromUint64Checked(uintVal); ok {
				return &converted
			}
			return nil
		}
		if floatVal, err := typed.Float64(); err == nil {
			if converted, ok := intFromFloat64Checked(floatVal); ok {
				return &converted
			}
			return nil
		}
		return nil
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return nil
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return nil
		}
		result := parsed
		return &result
	default:
		return nil
	}
}

func intFromInt64Checked(value int64) (int, bool) {
	if value < minPlatformInt64() || value > maxPlatformInt64() {
		return 0, false
	}
	return int(value), true
}

func intFromUint64Checked(value uint64) (int, bool) {
	if value > uint64(maxPlatformInt64()) {
		return 0, false
	}
	return int(value), true
}

func intFromFloat64Checked(value float64) (int, bool) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, false
	}
	if value != math.Trunc(value) {
		return 0, false
	}
	if value < float64(minPlatformInt64()) || value > float64(maxPlatformInt64()) {
		return 0, false
	}
	result := int(value)
	if float64(result) != value {
		return 0, false
	}
	return result, true
}

func maxPlatformInt64() int64 {
	return int64(^uint(0) >> 1)
}

func minPlatformInt64() int64 {
	return -maxPlatformInt64() - 1
}

func readAnyStringSlice(value any) []string {
	if value == nil {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			trimmed := strings.TrimSpace(item)
			if trimmed == "" {
				continue
			}
			result = append(result, trimmed)
		}
		return result
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			trimmed := readAnyString(item)
			if trimmed == "" {
				continue
			}
			result = append(result, trimmed)
		}
		return result
	default:
		return nil
	}
}

func readAnyMap(value any) map[string]any {
	if value == nil {
		return nil
	}
	switch typed := value.(type) {
	case map[string]any:
		return cloneAnyMapShallow(typed)
	default:
		return nil
	}
}

package transport

import "time"

func stringPointer(value string) *string {
	return &value
}

func boolPointer(value bool) *bool {
	return &value
}

func intPointer(value int) *int {
	return &value
}

func int64Pointer(value int64) *int64 {
	return &value
}

func timePointer(value time.Time) *time.Time {
	return &value
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func derefBool(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

func derefInt(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func derefInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func derefTime(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return *value
}

func optionalStringPointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func optionalIntPointer(value int) *int {
	if value == 0 {
		return nil
	}
	return &value
}

func optionalInt64Pointer(value int64) *int64 {
	if value == 0 {
		return nil
	}
	return &value
}

func optionalTimePointer(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}

func optionalAnyMapPointer(value map[string]any) *map[string]any {
	if len(value) == 0 {
		return nil
	}
	clone := make(map[string]any, len(value))
	for key, item := range value {
		clone[key] = item
	}
	return &clone
}

func fromOpenAPIAnyMap(value *map[string]any) map[string]any {
	if value == nil {
		return nil
	}
	clone := make(map[string]any, len(*value))
	for key, item := range *value {
		clone[key] = item
	}
	return clone
}

func toOpenAPIStringSlicePointer(values []string) *[]string {
	copyValues := append([]string(nil), values...)
	return &copyValues
}

func fromOpenAPIStringSlicePointer(values *[]string) []string {
	if values == nil {
		return nil
	}
	return append([]string(nil), (*values)...)
}

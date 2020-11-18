package cf

import (
	"fmt"
)

func MapIToMapS(in map[interface{}]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range in {
		result[fmt.Sprintf("%v", k)] = CleanUpMapValue(v)
	}
	return result
}

func CleanUpInterfaceArray(in []interface{}) []interface{} {
	result := make([]interface{}, len(in))
	for i, v := range in {
		result[i] = CleanUpMapValue(v)
	}
	return result
}

func CleanUpMapValue(v interface{}) interface{} {
	switch v := v.(type) {
	case []interface{}:
		return CleanUpInterfaceArray(v)

	case map[interface{}]interface{}:
		return MapIToMapS(v)

	default:
		return v
	}
}

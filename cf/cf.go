package cf

import (
	"fmt"
	"github.com/pkg/errors"
	"reflect"
)

func Load(data map[string]interface{}, cf interface{}) error {
	cfV := reflect.ValueOf(cf)
	if cfV.Kind() == reflect.Ptr {
		cfV = cfV.Elem()
	}
	if cfV.Kind() != reflect.Struct {
		return errors.Errorf("cf type [%s] not struct", cfV.Type())
	}
	for i := 0; i < cfV.NumField(); i++ {
		if cfV.Field(i).CanInterface() {
			key := keyName(cfV.Type().Field(i))
			if v, found := data[key]; found {
				if cfV.Field(i).CanSet() {
					switch cfV.Field(i).Interface().(type) {
					case int:
						if j, ok := v.(int); ok {
							cfV.Field(i).SetInt(int64(j))
						} else {
							return errors.Errorf("field '%s' type mismatch, got [%s], expected [%s]", key, reflect.TypeOf(v), cfV.Field(i).Type())
						}

					case float64:
						if f, ok := v.(float64); ok {
							cfV.Field(i).SetFloat(f)
						} else {
							return errors.Errorf("field '%s' type mismatch, got [%s], expected [%s]", key, reflect.TypeOf(v), cfV.Field(i).Type())
						}

					case bool:
						if b, ok := v.(bool); ok {
							cfV.Field(i).SetBool(b)
						} else {
							return errors.Errorf("field '%s' type mismatch, got [%s], expected [%s]", key, reflect.TypeOf(v), cfV.Field(i).Type())
						}

					case string:
						if s, ok := v.(string); ok {
							cfV.Field(i).SetString(s)
						} else {
							return errors.Errorf("field '%s' type mismatch, got [%s], expected [%s]", key, reflect.TypeOf(v), cfV.Field(i).Type())
						}

					default:
						return errors.Errorf("unsupported field type [%s]", cfV.Field(i).Type())
					}
				}
			}
		}
	}
	return nil
}

func Dump(label string, cf interface{}) string {
	cfV := reflect.ValueOf(cf)
	if cfV.Kind() == reflect.Ptr {
		cfV = cfV.Elem()
	}
	if cfV.Kind() != reflect.Struct {
		return ""
	}
	out := label + " {\n"
	format := fmt.Sprintf("\t%%-%ds %%v\n", maxKeyLength(cfV))
	for i := 0; i < cfV.NumField(); i++ {
		if cfV.Field(i).CanInterface() {
			key := keyName(cfV.Type().Field(i))
			out += fmt.Sprintf(format, key, cfV.Field(i).Interface())
		}
	}
	out += "}\n"
	return out
}

func keyName(v reflect.StructField) string {
	key := v.Name
	tag := v.Tag.Get("cf")
	if tag != "" {
		key = tag
	}
	return key
}

func maxKeyLength(cfV reflect.Value) int {
	maxKeyLength := 0
	for i := 0; i < cfV.NumField(); i++ {
		key := keyName(cfV.Type().Field(i))
		keyLength := len(key)
		if keyLength > maxKeyLength {
			maxKeyLength = keyLength
		}
	}
	return maxKeyLength
}

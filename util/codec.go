package util

import (
	"fmt"
	"reflect"
	"strconv"
)

func ParseInt(value string) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}

func ParseIdToString(value interface{}) (string, error) {
	switch v := value.(type) {
	case nil:
		return "", fmt.Errorf("id is nil")
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	case fmt.Stringer:
		return v.String(), nil
	case int:
		return strconv.Itoa(v), nil
	case int8:
		return strconv.FormatInt(int64(v), 10), nil
	case int16:
		return strconv.FormatInt(int64(v), 10), nil
	case int32:
		return strconv.FormatInt(int64(v), 10), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case uint:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint64:
		return strconv.FormatUint(v, 10), nil
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(v), nil
	default:
		return "", fmt.Errorf("unsupported id type: %T", value)
	}
}

func DecodeMapToStruct(input map[string]interface{}, target interface{}, tagName string) error {
	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Ptr || value.IsNil() {
		return fmt.Errorf("target must be a non-nil pointer")
	}

	elem := value.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("target must point to a struct")
	}

	elemType := elem.Type()
	for i := 0; i < elem.NumField(); i++ {
		fieldValue := elem.Field(i)
		fieldType := elemType.Field(i)
		if !fieldValue.CanSet() {
			continue
		}

		key := fieldType.Name
		if tagName != "" {
			if tagValue := fieldType.Tag.Get(tagName); tagValue != "" && tagValue != "-" {
				key = tagValue
			}
		}

		raw, ok := input[key]
		if !ok {
			continue
		}

		if err := assignValue(fieldValue, raw); err != nil {
			return fmt.Errorf("decode field %s: %w", fieldType.Name, err)
		}
	}

	return nil
}

func assignValue(target reflect.Value, raw interface{}) error {
	if raw == nil {
		return nil
	}

	rawValue := reflect.ValueOf(raw)
	if rawValue.Type().AssignableTo(target.Type()) {
		target.Set(rawValue)
		return nil
	}

	if rawValue.Type().ConvertibleTo(target.Type()) {
		target.Set(rawValue.Convert(target.Type()))
		return nil
	}

	switch target.Kind() {
	case reflect.String:
		target.SetString(fmt.Sprint(raw))
		return nil
	case reflect.Bool:
		switch v := raw.(type) {
		case bool:
			target.SetBool(v)
			return nil
		case string:
			parsed, err := strconv.ParseBool(v)
			if err != nil {
				return err
			}
			target.SetBool(parsed)
			return nil
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch v := raw.(type) {
		case string:
			parsed, err := strconv.ParseInt(v, 10, target.Type().Bits())
			if err != nil {
				return err
			}
			target.SetInt(parsed)
			return nil
		case float32:
			target.SetInt(int64(v))
			return nil
		case float64:
			target.SetInt(int64(v))
			return nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch v := raw.(type) {
		case string:
			parsed, err := strconv.ParseUint(v, 10, target.Type().Bits())
			if err != nil {
				return err
			}
			target.SetUint(parsed)
			return nil
		case float32:
			target.SetUint(uint64(v))
			return nil
		case float64:
			target.SetUint(uint64(v))
			return nil
		}
	case reflect.Float32, reflect.Float64:
		switch v := raw.(type) {
		case string:
			parsed, err := strconv.ParseFloat(v, target.Type().Bits())
			if err != nil {
				return err
			}
			target.SetFloat(parsed)
			return nil
		}
	}

	return fmt.Errorf("cannot assign %T to %s", raw, target.Type())
}

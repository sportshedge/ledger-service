package gotypes

import (
	"encoding/json"
	"reflect"
)

// isJSONMap checks if the input's internal type is map that is json formatted
//
// mapValType is the desired type for map's value, code will check if the type of the value is your desired type.
func isJSONMap(input string, mapValType reflect.Kind) bool {
	data := []byte(input)
	var err error

	switch mapValType {
	case reflect.Map:
		var m map[string]interface{}
		err = json.Unmarshal(data, &m)
		return err == nil && len(m) > 0 && isMapOfMap(m)
	case reflect.Bool:
		var m map[string]bool
		err = json.Unmarshal(data, &m)
		return err == nil && len(m) > 0 && isMapOfBool(m)
	case reflect.String:
		var m map[string]string
		err = json.Unmarshal(data, &m)
		return err == nil && len(m) > 0 && isMapOfString(m)
	}

	return false
}

// isJSONMapOfMap checks if map's value is a specific type of map with mapValType as the val
//
// Example:
// 		isJSONMapOfMap(`{"a": {"b": "c"}}`, reflect.Bool)
//	returns:
//		true
//
// Note: If you want to check if input is map of maps, then simply use isJSONMap and pass `reflect.Map` as the 2nd param.
func isJSONMapOfMap(input string, mapValType reflect.Kind) bool {
	data := []byte(input)
	var err error

	switch mapValType {
	case reflect.Bool:
		var m map[string]map[string]bool
		err = json.Unmarshal(data, &m)
		return err == nil && len(m) > 0
	case reflect.String:
		var m map[string]map[string]string
		err = json.Unmarshal(data, &m)
		return err == nil && len(m) > 0
	}

	return false
}

// isMapOfMap checks if provided data is a go map or not, it doesn't check for json map.
func isMapOfMap(data interface{}) bool {
	// Use reflection to get the underlying type of the interface
	dataType := reflect.TypeOf(data)

	// Check if the underlying type is a map
	if dataType.Kind() != reflect.Map {
		return false
	}

	// Check if the map's key type is string
	if dataType.Key().Kind() != reflect.String {
		return false
	}

	// Attempt to create an instance of the interface
	// and check if it's a map of maps
	//elemValue := reflect.New(dataType.Elem()).Elem()

	actualValue, ok := data.(map[string]interface{})
	if !ok {
		return false
	}

	// Now, check if the actual value is a map
	for _, value := range actualValue {
		if _, isMap := value.(map[string]interface{}); !isMap {
			return false
		}
	}

	return true
}

func isMapOfString(data interface{}) bool {
	if reflect.TypeOf(data).Kind() == reflect.Map {
		return reflect.TypeOf(data).Key().Kind() == reflect.String &&
			reflect.TypeOf(data).Elem().Kind() == reflect.String
	}
	if reflect.TypeOf(data).Kind() == reflect.String {
		var resultMap map[string]string
		jsonString := data.(string)

		if err := json.Unmarshal([]byte(jsonString), &resultMap); err == nil {
			return true
		}

	}

	return false
}

func isMapOfBool(data interface{}) bool {
	if reflect.TypeOf(data).Kind() == reflect.Map {
		return reflect.TypeOf(data).Key().Kind() == reflect.String &&
			reflect.TypeOf(data).Elem().Kind() == reflect.Bool
	}
	if reflect.TypeOf(data).Kind() == reflect.String {
		var resultMap map[string]bool
		jsonString := data.(string)

		if err := json.Unmarshal([]byte(jsonString), &resultMap); err == nil {
			return true
		}
	}

	return false
}

func IsMapBool(data string) bool {
	return isJSONMap(data, reflect.Bool)
}

func IsMapString(data string) bool {
	return isMapOfString(data)
}

func IsMapMapString(data string) bool {
	if isMapOfMap(data) || isJSONMap(data, reflect.Map) {
		return true
	}

	if !isMapOfMap(data) && !isJSONMap(data, reflect.Map) {
		return false
	}

	mapValue := reflect.ValueOf(data)
	for _, key := range mapValue.MapKeys() {
		if !isMapOfString(mapValue.MapIndex(key).Interface()) {
			stringValue, ok := mapValue.MapIndex(key).Interface().(string)
			if !ok {
				return false
			}
			if !isJSONMap(stringValue, reflect.String) {
				return false
			}
		}
	}

	return true
}

func IsMapMapBool(data string) bool {
	if isMapOfMap(data) || isJSONMapOfMap(data, reflect.Bool) {
		return true
	}

	if !isMapOfMap(data) && !isJSONMapOfMap(data, reflect.Bool) {
		return false
	}
	mapValue := reflect.ValueOf(data)
	for _, key := range mapValue.MapKeys() {
		if !isMapOfBool(mapValue.MapIndex(key).Interface()) {
			stringValue, ok := mapValue.MapIndex(key).Interface().(string)
			if !ok {
				return false
			}
			if !isJSONMap(stringValue, reflect.Bool) {
				return false
			}
		}
	}

	return true
}

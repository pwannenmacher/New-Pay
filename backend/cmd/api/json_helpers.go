package main

import (
	"encoding/json"
	"net/http"
	"reflect"
)

// JSONResponse sends a JSON response and ensures slices are never null
//
// IMPORTANT: This helper solves a common Go/JSON issue where nil slices are encoded as "null"
// instead of "[]". This causes problems in TypeScript/JavaScript frontends that expect arrays.
//
// Always use this function instead of json.NewEncoder(w).Encode() to avoid null slice issues.
//
// Example:
//
//	JSONResponse(w, myData)  // ✅ Correct - nil slices become []
//	json.NewEncoder(w).Encode(myData)  // ❌ Wrong - nil slices become null
func JSONResponse(w http.ResponseWriter, data interface{}) error {
	// Normalize the data to ensure slices are empty arrays instead of null
	normalized := normalizeSlices(data)

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(normalized)
}

// normalizeSlices recursively ensures all nil slices become empty slices
func normalizeSlices(data interface{}) interface{} {
	if data == nil {
		return data
	}

	v := reflect.ValueOf(data)

	// Handle pointers
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return data
		}
		elem := v.Elem()
		normalized := normalizeSlices(elem.Interface())

		// Create a new pointer to the normalized value
		result := reflect.New(elem.Type())
		result.Elem().Set(reflect.ValueOf(normalized))
		return result.Interface()
	}

	// Handle slices
	if v.Kind() == reflect.Slice {
		if v.IsNil() {
			// Return empty slice of the same type
			return reflect.MakeSlice(v.Type(), 0, 0).Interface()
		}

		// Normalize each element in the slice
		result := reflect.MakeSlice(v.Type(), v.Len(), v.Cap())
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			normalized := normalizeSlices(elem.Interface())
			result.Index(i).Set(reflect.ValueOf(normalized))
		}
		return result.Interface()
	}

	// Handle structs
	if v.Kind() == reflect.Struct {
		result := reflect.New(v.Type()).Elem()
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			if field.CanInterface() {
				normalized := normalizeSlices(field.Interface())
				if result.Field(i).CanSet() {
					result.Field(i).Set(reflect.ValueOf(normalized))
				}
			}
		}
		return result.Interface()
	}

	return data
}

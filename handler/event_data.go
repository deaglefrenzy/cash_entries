package handler

import (
	"fmt"
	"reflect"
	"time"

	"github.com/googleapis/google-cloudevents-go/cloud/firestoredata"
)

// To is a generic function that maps Firestore event data to any Go struct
// that uses `firestore:"..."` tags. It works similarly to the official client library's
// DocumentSnapshot.DataTo() method.
//
// Parameters:
//
//	data: The map of fields from the Firestore DocumentEventData.Value.
//	v: A pointer to the struct that you want to populate.
//
// Returns an error if the input is not a pointer to a struct or if a mapping error occurs.
func To(data map[string]*firestoredata.Value, v interface{}) error {
	// Ensure the input 'v' is a pointer, as we need to modify the underlying value.
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("models.EventDataTo: input must be a pointer, got %T", v)
	}

	// Dereference the pointer to get the actual struct value.
	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("models.EventDataTo: input must be a pointer to a struct, got %T", v)
	}

	return mapToStruct(data, val)
}

// mapToStruct is the core recursive function that populates a struct value.
func mapToStruct(data map[string]*firestoredata.Value, val reflect.Value) error {
	// Get the type of the struct we are populating.
	typ := val.Type()

	// Iterate over the fields of the struct.
	for i := 0; i < typ.NumField(); i++ {
		fieldTyp := typ.Field(i)
		fieldVal := val.Field(i)

		// Skip unexported fields, as we cannot set them.
		if !fieldVal.CanSet() {
			continue
		}

		// Get the firestore tag for the field name.
		tag := fieldTyp.Tag.Get("firestore")
		if tag == "" || tag == "-" {
			continue // Skip fields without a tag.
		}

		// Find the corresponding data from the input map.
		firestoreValue, ok := data[tag]
		if !ok {
			continue // No data for this field, leave it as its zero value.
		}

		// Set the field's value using the data from Firestore.
		if err := setFieldValue(fieldVal, firestoreValue); err != nil {
			return fmt.Errorf("failed to set field '%s': %w", fieldTyp.Name, err)
		}
	}

	return nil
}

// setFieldValue converts a *firestoredata.Value into the appropriate Go type and sets it
// on the given reflect.Value for a struct field.
func setFieldValue(fieldVal reflect.Value, firestoreValue *firestoredata.Value) error {
	if fieldVal.Kind() == reflect.Ptr {
		// If the field is a pointer, we need to create a new instance of the underlying
		// type and set the pointer to it before populating it.
		if fieldVal.IsNil() {
			fieldVal.Set(reflect.New(fieldVal.Type().Elem()))
		}
		// Now work with the dereferenced value.
		fieldVal = fieldVal.Elem()
	}

	// Handle the case where the firestore value is NULL.
	if _, ok := firestoreValue.ValueType.(*firestoredata.Value_NullValue); ok {
		// Field is already its zero value, so we do nothing.
		return nil
	}

	switch fieldVal.Kind() {
	case reflect.String:
		fieldVal.SetString(firestoreValue.GetStringValue())

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fieldVal.SetInt(firestoreValue.GetIntegerValue())

	case reflect.Float32, reflect.Float64:
		// Firestore numbers can be integer or double, so we handle both.
		var floatVal float64
		if dVal, ok := firestoreValue.ValueType.(*firestoredata.Value_DoubleValue); ok {
			floatVal = dVal.DoubleValue
		} else if iVal, ok := firestoreValue.ValueType.(*firestoredata.Value_IntegerValue); ok {
			floatVal = float64(iVal.IntegerValue)
		}
		fieldVal.SetFloat(floatVal)

	case reflect.Bool:
		fieldVal.SetBool(firestoreValue.GetBooleanValue())

	case reflect.Struct:
		// Handle time.Time as a special case of struct.
		if fieldVal.Type() == reflect.TypeOf(time.Time{}) {
			if ts := firestoreValue.GetTimestampValue(); ts != nil {
				fieldVal.Set(reflect.ValueOf(ts.AsTime()))
			}
			return nil
		}

		// For other structs, we recurse.
		if mapVal := firestoreValue.GetMapValue(); mapVal != nil {
			return mapToStruct(mapVal.Fields, fieldVal)
		}
		return fmt.Errorf("expected a map for nested struct, but got %T", firestoreValue.ValueType)

	case reflect.Map:
		mapVal := firestoreValue.GetMapValue()
		if mapVal == nil {
			return fmt.Errorf("expected a map to populate a map field, but got %T", firestoreValue.ValueType)
		}

		// Create a new map of the correct type.
		mapType := fieldVal.Type()
		newMap := reflect.MakeMap(mapType)
		elemType := mapType.Elem()

		for key, val := range mapVal.Fields {
			// Create a new element of the map's value type.
			newElem := reflect.New(elemType).Elem()

			// Recursively set the value of the new map element.
			if err := setFieldValue(newElem, val); err != nil {
				return err
			}

			// Add the new element to the map.
			newMap.SetMapIndex(reflect.ValueOf(key), newElem)
		}
		fieldVal.Set(newMap)

	default:
		return fmt.Errorf("unsupported field type: %s", fieldVal.Kind())
	}

	return nil
}

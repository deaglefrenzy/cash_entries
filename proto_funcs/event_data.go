package proto_funcs

import (
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/googleapis/google-cloudevents-go/cloud/firestoredata"
	"google.golang.org/genproto/googleapis/type/latlng"
)

var (
	typeOfByteSlice = reflect.TypeOf([]byte{})
	typeOfGoTime    = reflect.TypeOf(time.Time{})
	typeOfLatLng    = reflect.TypeOf(latlng.LatLng{})
	typeOfUUID      = reflect.TypeOf(uuid.UUID{})
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
func FirestoreDataTo(data map[string]*firestoredata.Value, v interface{}) error {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return fmt.Errorf("FirestoreDataTo: expected non-nil pointer to struct, got %T", v)
	}

	elem := val.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("FirestoreDataTo: expected pointer to struct, got pointer to %s", elem.Kind())
	}

	return mapToStruct(data, elem)
}

// mapToStruct is the core recursive function that populates a struct value.
func mapToStruct(data map[string]*firestoredata.Value, val reflect.Value) error {
	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		fieldTyp := typ.Field(i)
		fieldVal := val.Field(i)

		if !fieldVal.CanSet() {
			continue
		}

		tag := fieldTyp.Tag.Get("firestore")
		if tag == "" || tag == "-" {
			continue
		}

		firestoreValue, ok := data[tag]
		if !ok {
			continue
		}

		if err := setFieldValue(fieldVal, firestoreValue); err != nil {
			return fmt.Errorf("failed to set field %s: %w", fieldTyp.Name, err)
		}
	}
	return nil
}

// setFieldValue converts a *firestoredata.Value into the appropriate Go type and sets it
// on the given reflect.Value for a struct field.
func setFieldValue(fieldVal reflect.Value, firestoreValue *firestoredata.Value) error {
	if firestoreValue == nil || firestoreValue.ValueType == nil {
		return nil // ignore nils
	}

	// Handle Firestore explicit nulls
	if _, ok := firestoreValue.ValueType.(*firestoredata.Value_NullValue); ok {
		// Leave as zero value, or nil if pointer
		if fieldVal.Kind() == reflect.Ptr {
			fieldVal.Set(reflect.Zero(fieldVal.Type()))
		}
		return nil
	}

	// Handle pointers
	if fieldVal.Kind() == reflect.Ptr {
		if fieldVal.IsNil() {
			fieldVal.Set(reflect.New(fieldVal.Type().Elem()))
		}
		return setFieldValue(fieldVal.Elem(), firestoreValue)
	}

	switch fieldVal.Type() {
	case typeOfGoTime:
		if ts := firestoreValue.GetTimestampValue(); ts != nil {
			fieldVal.Set(reflect.ValueOf(ts.AsTime()))
		}
		return nil

	case typeOfByteSlice:
		if bs := firestoreValue.GetBytesValue(); bs != nil {
			fieldVal.SetBytes(bs)
		}
		return nil

	case typeOfUUID:
		s := firestoreValue.GetStringValue()
		parsed, err := uuid.Parse(s)
		if err != nil {
			return fmt.Errorf("invalid UUID: %v", err)
		}
		fieldVal.Set(reflect.ValueOf(parsed))
		return nil

	case typeOfLatLng:
		if latlngVal := firestoreValue.GetGeoPointValue(); latlngVal != nil {
			ll := &latlng.LatLng{
				Latitude:  latlngVal.Latitude,
				Longitude: latlngVal.Longitude,
			}
			fieldVal.Set(reflect.ValueOf(ll))
		}
		return nil
	}

	// Handle common kinds
	switch fieldVal.Kind() {
	case reflect.String:
		fieldVal.SetString(firestoreValue.GetStringValue())

	case reflect.Bool:
		fieldVal.SetBool(firestoreValue.GetBooleanValue())

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fieldVal.SetInt(firestoreValue.GetIntegerValue())

	case reflect.Float32, reflect.Float64:
		switch v := firestoreValue.ValueType.(type) {
		case *firestoredata.Value_DoubleValue:
			fieldVal.SetFloat(v.DoubleValue)
		case *firestoredata.Value_IntegerValue:
			fieldVal.SetFloat(float64(v.IntegerValue))
		default:
			return fmt.Errorf("unsupported numeric type %T for float field", v)
		}

	case reflect.Slice:
		arr := firestoreValue.GetArrayValue()
		if arr == nil {
			return fmt.Errorf("expected array value for slice field")
		}
		elemType := fieldVal.Type().Elem()
		slice := reflect.MakeSlice(fieldVal.Type(), 0, len(arr.Values))
		for _, item := range arr.Values {
			elem := reflect.New(elemType).Elem()
			if err := setFieldValue(elem, item); err != nil {
				return err
			}
			slice = reflect.Append(slice, elem)
		}
		fieldVal.Set(slice)

	case reflect.Map:
		mapVal := firestoreValue.GetMapValue()
		if mapVal == nil {
			return fmt.Errorf("expected map value for map field")
		}
		newMap := reflect.MakeMap(fieldVal.Type())
		elemType := fieldVal.Type().Elem()
		for key, val := range mapVal.Fields {
			elem := reflect.New(elemType).Elem()
			if err := setFieldValue(elem, val); err != nil {
				return err
			}
			newMap.SetMapIndex(reflect.ValueOf(key), elem)
		}
		fieldVal.Set(newMap)

	case reflect.Struct:
		mapVal := firestoreValue.GetMapValue()
		if mapVal == nil {
			return fmt.Errorf("expected map for struct field, got %T", firestoreValue.ValueType)
		}
		return mapToStruct(mapVal.Fields, fieldVal)

	default:
		return fmt.Errorf("unsupported kind %s for field", fieldVal.Kind())
	}
	return nil
}

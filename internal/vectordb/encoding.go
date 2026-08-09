/*
 * Copyright 2026 Holger de Carne
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package vectordb

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/qdrant/go-client/qdrant"
)

type Point interface {
	GetId() *qdrant.PointId
	GetPayload() map[string]*qdrant.Value
}

func EncodeToPoint(v any) (*qdrant.PointStruct, error) {
	vValue, err := safeValueOf(v)
	if err != nil {
		return nil, err
	}
	structValue := vValue.Elem()
	structType := structValue.Type()
	numField := structValue.NumField()
	point := &qdrant.PointStruct{}
	payload := make(map[string]any)
	for i := range numField {
		field := structType.Field(i)
		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		if tag == "id" {
			point.Id = qdrant.NewID(structValue.Field(i).String())
		} else {
			field := structValue.Field(i)
			switch field.Kind() {
			case reflect.Slice, reflect.Array:
				payload[tag] = encodeSliceOrArray(field)
			default:
				payload[tag] = encodeInterface(field)
			}
		}
	}
	point.Payload = qdrant.NewValueMap(payload)
	return point, nil
}

func encodeSliceOrArray(field reflect.Value) []any {
	fieldLen := field.Len()
	value := make([]any, fieldLen)
	for i := range fieldLen {
		value[i] = field.Index(i).Interface()
	}
	return value
}

func encodeInterface(field reflect.Value) any {
	switch v := field.Interface().(type) {
	case time.Time:
		return encodeDatetime(v)
	default:
		return v
	}
}

func DecodeFromPoint(v any, point Point) error {
	vValue, err := safeValueOf(v)
	if err != nil {
		return err
	}
	structValue := vValue.Elem()
	structType := structValue.Type()
	numField := structValue.NumField()
	for i := range numField {
		typeField := structType.Field(i)
		tag := typeField.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		field := structValue.Field(i)
		pointID := point.GetId()
		pointPayload := point.GetPayload()
		if tag == "id" {
			err = decodeUUIDValue(&field, pointID)
		} else if payloadValue, ok := pointPayload[tag]; ok {
			err = decodePayloadValue(&field, payloadValue)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func decodeUUIDValue(field *reflect.Value, value *qdrant.PointId) error {
	field.Set(reflect.ValueOf(value.GetUuid()))
	return nil
}

func decodePayloadValue(field *reflect.Value, value *qdrant.Value) error {
	switch value.Kind.(type) {
	case *qdrant.Value_StringValue:
		return decodeStringValue(field, value)
	case *qdrant.Value_IntegerValue:
		return decodeIntegerValue(field, value)
	case *qdrant.Value_DoubleValue:
		return decodeDoubleValue(field, value)
	case *qdrant.Value_BoolValue:
		return decodeBoolValue(field, value)
	case *qdrant.Value_ListValue:
		return decodeListValue(field, value)
	default:
		return fmt.Errorf("unexpected value type (%T)", value.Kind)
	}
}

func decodeStringValue(field *reflect.Value, value *qdrant.Value) error {
	stringValue := value.GetStringValue()
	switch field.Interface().(type) {
	case time.Time:
		timeValue, err := decodeDatetime(stringValue)
		if err != nil {
			return fmt.Errorf("invalid Datetime value '%s' (cause: %w)", stringValue, err)
		}
		field.Set(reflect.ValueOf(timeValue))
	default:
		field.Set(reflect.ValueOf(stringValue))
	}
	return nil
}

func decodeIntegerValue(field *reflect.Value, value *qdrant.Value) error {
	field.Set(reflect.ValueOf(value.GetIntegerValue()))
	return nil
}

func decodeDoubleValue(field *reflect.Value, value *qdrant.Value) error {
	field.Set(reflect.ValueOf(value.GetDoubleValue()))
	return nil
}

func decodeBoolValue(field *reflect.Value, value *qdrant.Value) error {
	field.Set(reflect.ValueOf(value.GetBoolValue()))
	return nil
}

func decodeListValue(field *reflect.Value, value *qdrant.Value) error {
	listValues := value.GetListValue().GetValues()
	fieldValues := make([]string, len(listValues))
	for i, listValue := range listValues {
		fieldValues[i] = listValue.GetStringValue()
	}
	field.Set(reflect.ValueOf(fieldValues))
	return nil
}

func safeValueOf(v any) (reflect.Value, error) {
	vValue := reflect.ValueOf(v)
	if vValue.Kind() != reflect.Ptr || vValue.Elem().Kind() != reflect.Struct {
		return vValue, errors.New("invalid target; must be pointer to struct")
	}
	return vValue, nil
}

func encodeDatetime(t time.Time) string {
	return t.Format(time.RFC3339)
}

func decodeDatetime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

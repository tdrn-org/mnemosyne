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
	"reflect"

	"github.com/qdrant/go-client/qdrant"
)

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
				payload[tag] = field.Interface()
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

func DecodeFromRetrievedPoint(v any, point *qdrant.RetrievedPoint) error {
	vValue, err := safeValueOf(v)
	if err != nil {
		return err
	}
	structValue := vValue.Elem()
	structType := structValue.Type()
	numField := structValue.NumField()
	for i := range numField {
		field := structType.Field(i)
		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		if tag == "id" {
			structValue.Field(i).Set(reflect.ValueOf(point.Id.GetUuid()))
		} else if payloadValue, ok := point.Payload[tag]; ok {
			switch payloadValue.Kind.(type) {
			case *qdrant.Value_StringValue:
				structValue.Field(i).Set(reflect.ValueOf(point.Payload[tag].GetStringValue()))
			case *qdrant.Value_IntegerValue:
				structValue.Field(i).Set(reflect.ValueOf(point.Payload[tag].GetIntegerValue()))
			case *qdrant.Value_BoolValue:
				structValue.Field(i).Set(reflect.ValueOf(point.Payload[tag].GetBoolValue()))
			}
		}
	}
	return nil
}

func DecodeFromScoredPoint(v any, point *qdrant.ScoredPoint) error {
	vValue, err := safeValueOf(v)
	if err != nil {
		return err
	}
	structValue := vValue.Elem()
	structType := structValue.Type()
	numField := structValue.NumField()
	for i := range numField {
		field := structType.Field(i)
		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		if tag == "id" {
			structValue.Field(i).Set(reflect.ValueOf(point.Id.GetUuid()))
		} else if payloadValue, ok := point.Payload[tag]; ok {
			switch payloadValue.Kind.(type) {
			case *qdrant.Value_StringValue:
				structValue.Field(i).Set(reflect.ValueOf(point.Payload[tag].GetStringValue()))
			case *qdrant.Value_IntegerValue:
				structValue.Field(i).Set(reflect.ValueOf(point.Payload[tag].GetIntegerValue()))
			case *qdrant.Value_BoolValue:
				structValue.Field(i).Set(reflect.ValueOf(point.Payload[tag].GetBoolValue()))
			}
		}
	}
	return nil
}

func safeValueOf(v any) (reflect.Value, error) {
	vValue := reflect.ValueOf(v)
	if vValue.Kind() != reflect.Ptr || vValue.Elem().Kind() != reflect.Struct {
		return vValue, errors.New("invalid target; must be pointer to struct")
	}
	return vValue, nil
}

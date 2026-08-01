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

package vectordb_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/mnemosyne/internal/vectordb"
)

func TestEncoding(t *testing.T) {
	v := &EncodingTestStruct{
		ID:          uuid.NewString(),
		StringValue: t.Name(),
		IntValue:    42,
		BoolValue:   true,
	}
	point, err := vectordb.EncodeToPoint(v)
	require.NoError(t, err)
	require.Equal(t, v.ID, point.Id.GetUuid())
	require.Equal(t, v.StringValue, point.Payload["string_value"].GetStringValue())
	require.Equal(t, v.IntValue, point.Payload["int_value"].GetIntegerValue())
	require.Equal(t, v.BoolValue, point.Payload["bool_value"].GetBoolValue())
}

func TestDecoding(t *testing.T) {
	v := &EncodingTestStruct{}
	point := &qdrant.RetrievedPoint{
		Id: qdrant.NewID(uuid.NewString()),
		Payload: qdrant.NewValueMap(map[string]any{
			"string_value": t.Name(),
			"int_value":    42,
			"bool_value":   true,
		}),
	}
	err := vectordb.DecodeFromRetrievedPoint(v, point)
	require.NoError(t, err)
	require.Equal(t, point.Id.GetUuid(), v.ID)
	require.Equal(t, point.Payload["string_value"].GetStringValue(), v.StringValue)
	require.Equal(t, point.Payload["int_value"].GetIntegerValue(), v.IntValue)
	require.Equal(t, point.Payload["bool_value"].GetBoolValue(), v.BoolValue)
}

type EncodingTestStruct struct {
	ID          string `json:"id"`
	StringValue string `json:"string_value"`
	IntValue    int64  `json:"int_value"`
	BoolValue   bool   `json:"bool_value"`
}

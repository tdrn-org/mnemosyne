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
	"time"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/mnemosyne/internal/domain"
	"github.com/tdrn-org/mnemosyne/internal/vectordb"
)

func TestEncoding(t *testing.T) {
	v := &EncodingTestStruct{
		ID:          uuid.NewString(),
		StringValue: t.Name(),
		IntValue:    42,
		DoubleValue: 0.42,
		BoolValue:   true,
		ListValue:   []string{"1st string", "2nd string"},
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
			"double_value": 0.42,
			"bool_value":   true,
			"list_value":   []any{"1st string", "2nd string"},
		}),
	}
	err := vectordb.DecodeFromPoint(v, point)
	require.NoError(t, err)
	require.Equal(t, point.Id.GetUuid(), v.ID)
	require.Equal(t, point.Payload["string_value"].GetStringValue(), v.StringValue)
	require.Equal(t, point.Payload["int_value"].GetIntegerValue(), v.IntValue)
	require.Equal(t, point.Payload["bool_value"].GetBoolValue(), v.BoolValue)
}

func TestMemoryEncodingOmitEmpty(t *testing.T) {
	never := &domain.Memory{
		ID:      uuid.NewString(),
		Content: "an emotional moment",
		Type:    "emotional",
		Trust:   1.0,
	}
	point, err := vectordb.EncodeToPoint(never)
	require.NoError(t, err)
	_, hasExpires := point.Payload["expires_at"]
	require.False(t, hasExpires, "zero ExpiresAt must be omitted (never)")
	_, hasLabels := point.Payload["labels"]
	require.False(t, hasLabels, "empty Labels must be omitted")

	expiring := &domain.Memory{
		ID:        uuid.NewString(),
		Content:   "a fact",
		Type:      "fact",
		Trust:     0.9,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Labels:    []string{"x", "y"},
	}
	point2, err := vectordb.EncodeToPoint(expiring)
	require.NoError(t, err)
	_, hasExpiresClean := point2.Payload["expires_at"]
	require.True(t, hasExpiresClean, "expires_at must be present under clean key")
	_, hasExpiresRaw := point2.Payload["expires_at,omitempty"]
	require.False(t, hasExpiresRaw, "payload key must not contain ',omitempty'")
	labels := point2.Payload["labels"].GetListValue().GetValues()
	require.Len(t, labels, 2)
}

type EncodingTestStruct struct {
	ID          string   `json:"id"`
	StringValue string   `json:"string_value"`
	IntValue    int64    `json:"int_value"`
	DoubleValue float64  `json:"double_value"`
	BoolValue   bool     `json:"bool_value"`
	ListValue   []string `json:"list_value"`
}

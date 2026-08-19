// Copyright 2026 Specter Ops, Inc.
//
// Licensed under the Apache License, Version 2.0
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0
package payload_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/specterops/chow/pkg/payload"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type jsonSchemaAssertion struct {
	name  string
	raw   string
	valid bool
}

func assertJSONSchema(t *testing.T, schema *jsonschema.Schema, assertions []jsonSchemaAssertion) {
	t.Helper()

	for _, assertion := range assertions {
		t.Run(assertion.name, func(t *testing.T) {
			var document any
			require.NoError(t, json.Unmarshal([]byte(assertion.raw), &document))

			err := schema.Validate(document)
			if assertion.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestNodeJSONSchemaContract(t *testing.T) {
	schema, err := payload.LoadSchema()
	require.NoError(t, err)

	assertJSONSchema(t, schema.NodeSchema, []jsonSchemaAssertion{
		{
			name:  "minimal node",
			raw:   `{"id":"node-1","kinds":["User"]}`,
			valid: true,
		},
		{
			name:  "primitive property values",
			raw:   `{"id":"node-1","kinds":["Device","Asset"],"properties":{"name":"alpha","score":1.5,"enabled":true,"labels":["a","b"],"ports":[1,2],"flags":[true,false]}}`,
			valid: true,
		},
		{
			name:  "lowercase alphanumeric underscore and hyphen property name",
			raw:   `{"id":"node-1","kinds":["Device"],"properties":{"lowercase_1-name":"alpha"}}`,
			valid: true,
		},
		{
			name:  "uppercase property name",
			raw:   `{"id":"node-1","kinds":["User"],"properties":{"DisplayName":"Alice"}}`,
			valid: false,
		},
		{
			name:  "punctuation in property name",
			raw:   `{"id":"node-1","kinds":["User"],"properties":{"display.name":"Alice"}}`,
			valid: false,
		},
		{
			name:  "empty property name",
			raw:   `{"id":"node-1","kinds":["User"],"properties":{"":"Alice"}}`,
			valid: false,
		},
		{
			name:  "128 character property name",
			raw:   fmt.Sprintf(`{"id":"node-1","kinds":["User"],"properties":{"%s":"Alice"}}`, strings.Repeat("a", 128)),
			valid: true,
		},
		{
			name:  "129 character property name",
			raw:   fmt.Sprintf(`{"id":"node-1","kinds":["User"],"properties":{"%s":"Alice"}}`, strings.Repeat("a", 129)),
			valid: false,
		},
		{
			name:  "null properties",
			raw:   `{"id":"node-1","kinds":["Location"],"properties":null}`,
			valid: true,
		},
		{
			name:  "missing id",
			raw:   `{"kinds":["User"]}`,
			valid: false,
		},
		{
			name:  "missing kinds",
			raw:   `{"id":"node-1"}`,
			valid: false,
		},
		{
			name:  "too many kinds",
			raw:   `{"id":"node-1","kinds":["A","B","C","D"]}`,
			valid: false,
		},
		{
			name:  "kind cannot use tag prefix",
			raw:   `{"id":"node-1","kinds":["Tag_Admin"]}`,
			valid: false,
		},
		{
			name:  "objectid property is reserved",
			raw:   `{"id":"node-1","kinds":["User"],"properties":{"objectid":"node-1"}}`,
			valid: false,
		},
		{
			name:  "property value cannot be an object",
			raw:   `{"id":"node-1","kinds":["User"],"properties":{"profile":{"name":"Alice"}}}`,
			valid: false,
		},
		{
			name:  "array property values must be homogeneous",
			raw:   `{"id":"node-1","kinds":["User"],"properties":{"items":["one",2]}}`,
			valid: false,
		},
	})
}

func TestEdgeJSONSchemaContract(t *testing.T) {
	schema, err := payload.LoadSchema()
	require.NoError(t, err)

	assertJSONSchema(t, schema.EdgeSchema, []jsonSchemaAssertion{
		{
			name:  "id endpoints",
			raw:   `{"start":{"value":"node-1"},"end":{"value":"node-2"},"kind":"RELATED","properties":{"since":"today","weight":1,"active":true,"labels":["a"]}}`,
			valid: true,
		},
		{
			name:  "lowercase alphanumeric underscore and hyphen property name",
			raw:   `{"start":{"value":"node-1"},"end":{"value":"node-2"},"kind":"RELATED","properties":{"lowercase_1-name":"alpha"}}`,
			valid: true,
		},
		{
			name:  "uppercase property name",
			raw:   `{"start":{"value":"node-1"},"end":{"value":"node-2"},"kind":"RELATED","properties":{"DisplayName":"Alice"}}`,
			valid: false,
		},
		{
			name:  "punctuation in property name",
			raw:   `{"start":{"value":"node-1"},"end":{"value":"node-2"},"kind":"RELATED","properties":{"display.name":"Alice"}}`,
			valid: false,
		},
		{
			name:  "empty property name",
			raw:   `{"start":{"value":"node-1"},"end":{"value":"node-2"},"kind":"RELATED","properties":{"":"Alice"}}`,
			valid: false,
		},
		{
			name:  "128 character property name",
			raw:   fmt.Sprintf(`{"start":{"value":"node-1"},"end":{"value":"node-2"},"kind":"RELATED","properties":{"%s":"Alice"}}`, strings.Repeat("a", 128)),
			valid: true,
		},
		{
			name:  "129 character property name",
			raw:   fmt.Sprintf(`{"start":{"value":"node-1"},"end":{"value":"node-2"},"kind":"RELATED","properties":{"%s":"Alice"}}`, strings.Repeat("a", 129)),
			valid: false,
		},
		{
			name:  "property matching endpoints",
			raw:   `{"start":{"match_by":"property","property_matchers":[{"key":"display_name-1","operator":"equals","value":"Alice"}]},"end":{"match_by":"property","property_matchers":[{"key":"name","operator":"equals","value":"Bob"}]},"kind":"connected_to","properties":null}`,
			valid: true,
		},
		{
			name:  "uppercase property matcher key",
			raw:   `{"start":{"match_by":"property","property_matchers":[{"key":"DisplayName","operator":"equals","value":"Alice"}]},"end":{"value":"node-2"},"kind":"RELATED"}`,
			valid: false,
		},
		{
			name:  "punctuation in property matcher key",
			raw:   `{"start":{"match_by":"property","property_matchers":[{"key":"display.name","operator":"equals","value":"Alice"}]},"end":{"value":"node-2"},"kind":"RELATED"}`,
			valid: false,
		},
		{
			name:  "empty property matcher key",
			raw:   `{"start":{"match_by":"property","property_matchers":[{"key":"","operator":"equals","value":"Alice"}]},"end":{"value":"node-2"},"kind":"RELATED"}`,
			valid: false,
		},
		{
			name:  "128 character property matcher key",
			raw:   fmt.Sprintf(`{"start":{"match_by":"property","property_matchers":[{"key":"%s","operator":"equals","value":"Alice"}]},"end":{"value":"node-2"},"kind":"RELATED"}`, strings.Repeat("a", 128)),
			valid: true,
		},
		{
			name:  "129 character property matcher key",
			raw:   fmt.Sprintf(`{"start":{"match_by":"property","property_matchers":[{"key":"%s","operator":"equals","value":"Alice"}]},"end":{"value":"node-2"},"kind":"RELATED"}`, strings.Repeat("a", 129)),
			valid: false,
		},
		{
			name:  "objectid property is allowed on edges",
			raw:   `{"start":{"value":"node-1"},"end":{"value":"node-2"},"kind":"RELATED","properties":{"objectid":"edge-1"}}`,
			valid: true,
		},
		{
			name:  "endpoint kind filter",
			raw:   `{"start":{"value":"node-1","kind":"User"},"end":{"value":"node-2","kind":"Computer"},"kind":"admin_to"}`,
			valid: true,
		},
		{
			name:  "missing start",
			raw:   `{"end":{"value":"node-2"},"kind":"RELATED"}`,
			valid: false,
		},
		{
			name:  "missing end",
			raw:   `{"start":{"value":"node-1"},"kind":"RELATED"}`,
			valid: false,
		},
		{
			name:  "missing kind",
			raw:   `{"start":{"value":"node-1"},"end":{"value":"node-2"}}`,
			valid: false,
		},
		{
			name:  "kind must be alphanumeric or underscore",
			raw:   `{"start":{"value":"node-1"},"end":{"value":"node-2"},"kind":"bad-kind"}`,
			valid: false,
		},
		{
			name:  "kind cannot use tag prefix",
			raw:   `{"start":{"value":"node-1"},"end":{"value":"node-2"},"kind":"TAG_Admin"}`,
			valid: false,
		},
		{
			name:  "reserved namespace is allowed in endpoint kind filters",
			raw:   `{"start":{"value":"node-1","kind":"tag_Admin"},"end":{"value":"node-2","kind":"TaG"},"kind":"RELATED"}`,
			valid: true,
		},
		{
			name:  "id endpoint requires value",
			raw:   `{"start":{"match_by":"id"},"end":{"value":"node-2"},"kind":"RELATED"}`,
			valid: false,
		},
		{
			name:  "property endpoint requires matchers",
			raw:   `{"start":{"match_by":"property"},"end":{"value":"node-2"},"kind":"RELATED"}`,
			valid: false,
		},
		{
			name:  "property endpoint cannot use value",
			raw:   `{"start":{"match_by":"property","value":"node-1","property_matchers":[{"key":"name","operator":"equals","value":"Alice"}]},"end":{"value":"node-2"},"kind":"RELATED"}`,
			valid: false,
		},
		{
			name:  "non-property endpoint cannot use matchers",
			raw:   `{"start":{"match_by":"id","value":"node-1","property_matchers":[{"key":"name","operator":"equals","value":"Alice"}]},"end":{"value":"node-2"},"kind":"RELATED"}`,
			valid: false,
		},
		{
			name:  "property matchers cannot be empty",
			raw:   `{"start":{"match_by":"property","property_matchers":[]},"end":{"value":"node-2"},"kind":"RELATED"}`,
			valid: false,
		},
		{
			name:  "property matcher operator must be equals",
			raw:   `{"start":{"match_by":"property","property_matchers":[{"key":"name","operator":"contains","value":"Alice"}]},"end":{"value":"node-2"},"kind":"RELATED"}`,
			valid: false,
		},
		{
			name:  "property matcher value must be primitive",
			raw:   `{"start":{"match_by":"property","property_matchers":[{"key":"name","operator":"equals","value":{"first":"Alice"}}]},"end":{"value":"node-2"},"kind":"RELATED"}`,
			valid: false,
		},
		{
			name:  "property value cannot be an object",
			raw:   `{"start":{"value":"node-1"},"end":{"value":"node-2"},"kind":"RELATED","properties":{"profile":{"name":"Alice"}}}`,
			valid: false,
		},
		{
			name:  "array property values must be homogeneous",
			raw:   `{"start":{"value":"node-1"},"end":{"value":"node-2"},"kind":"RELATED","properties":{"items":["one",2]}}`,
			valid: false,
		},
	})
}

func TestMetadataJSONSchemaContract(t *testing.T) {
	schema, err := payload.LoadSchema()
	require.NoError(t, err)

	assertJSONSchema(t, schema.MetaSchema, []jsonSchemaAssertion{
		{
			name:  "empty metadata",
			raw:   `{}`,
			valid: true,
		},
		{
			name:  "source kind",
			raw:   `{"source_kind":"hellobase"}`,
			valid: true,
		},
		{
			name:  "null source kind",
			raw:   `{"source_kind":null}`,
			valid: true,
		},
		{
			name:  "source kind must be string or null",
			raw:   `{"source_kind":1}`,
			valid: false,
		},
		{
			name:  "additional properties are not allowed",
			raw:   `{"source_kind":"hellobase","extra":"value"}`,
			valid: false,
		},
	})
}

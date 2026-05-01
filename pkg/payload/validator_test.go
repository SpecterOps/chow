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
	"strings"
	"testing"

	"github.com/specterops/chow/pkg/ingest"
	"github.com/specterops/chow/pkg/payload"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var emptyValidationReport = payload.ValidationReport{CriticalErrors: []payload.CriticalError{}, ValidationErrors: []payload.ValidationError{}}

type parseAndValidateAssertion struct {
	name               string
	payload            string
	expectedParsedData payload.ParsedData
	errValidationFunc  func(t *testing.T, report payload.ValidationReport, err error)
}

func Test_ParseAndValidate(t *testing.T) {
	assertions := []parseAndValidateAssertion{
		// OpenGraph payload tests
		{
			name:               "successful opengraph payload",
			payload:            `{"metadata":{},"graph":{"nodes":[]}}`,
			expectedParsedData: payload.ParsedData{PayloadType: ingest.DataTypeOpenGraph},
			errValidationFunc: func(t *testing.T, report payload.ValidationReport, err error) {
				assert.Equal(t, emptyValidationReport, report)
				assert.NoError(t, err)
			},
		},
		{
			name:               "successful opengraph payload with no metadata",
			payload:            `{"graph":{"nodes":[]}}`,
			expectedParsedData: payload.ParsedData{PayloadType: ingest.DataTypeOpenGraph},
			errValidationFunc: func(t *testing.T, report payload.ValidationReport, err error) {
				assert.Equal(t, emptyValidationReport, report)
				assert.NoError(t, err)
			},
		},
		{
			name:               "successful opengraph metadata",
			payload:            `{"metadata":{"source_kind":"hellobase"},"graph":{"nodes":[]}}`,
			expectedParsedData: payload.ParsedData{PayloadType: ingest.DataTypeOpenGraph, OpengraphData: payload.ParsedOpenGraphData{Metadata: ingest.OpengraphMetadata{SourceKind: "hellobase"}}},
			errValidationFunc: func(t *testing.T, report payload.ValidationReport, err error) {
				assert.Equal(t, emptyValidationReport, report)
				assert.NoError(t, err)
			},
		},
		{
			name:               "successful opengraph payload with $schema",
			payload:            `{"$schema":"test","metadata":{"source_kind":"hellobase"},"graph":{"nodes":[]}}`,
			expectedParsedData: payload.ParsedData{PayloadType: ingest.DataTypeOpenGraph, OpengraphData: payload.ParsedOpenGraphData{Metadata: ingest.OpengraphMetadata{SourceKind: "hellobase"}}},
			errValidationFunc: func(t *testing.T, report payload.ValidationReport, err error) {
				assert.Equal(t, emptyValidationReport, report)
				assert.NoError(t, err)
			},
		},
		{
			name:               "successful opengraph payload with node",
			payload:            `{"metadata":{"source_kind":"hellobase"},"graph":{"nodes":[{"id":"TESTNODE","kinds":["User"],"properties":{"items":["hi"]}}]}}`,
			expectedParsedData: payload.ParsedData{PayloadType: ingest.DataTypeOpenGraph, OpengraphData: payload.ParsedOpenGraphData{Metadata: ingest.OpengraphMetadata{SourceKind: "hellobase"}, NodesValidated: 1}},
			errValidationFunc: func(t *testing.T, report payload.ValidationReport, err error) {
				assert.Equal(t, emptyValidationReport, report)
				assert.NoError(t, err)
			},
		},
		{
			name:               "unsuccessful opengraph payload, node id validation error",
			payload:            `{"metadata":{"source_kind":"hellobase"},"graph":{"nodes":[{"id":1,"kinds":["User"]}]}}`,
			expectedParsedData: payload.ParsedData{PayloadType: ingest.DataTypeOpenGraph, OpengraphData: payload.ParsedOpenGraphData{Metadata: ingest.OpengraphMetadata{SourceKind: "hellobase"}, NodesValidated: 1}},
			errValidationFunc: func(t *testing.T, report payload.ValidationReport, err error) {
				assert.ErrorIs(t, err, payload.ErrValidationErrors)

				assert.ElementsMatch(t, report.ValidationErrors, []payload.ValidationError{
					{
						Location:  "/graph/nodes[0]",
						RawObject: `{"id":1,"kinds":["User"]}`,
						Errors:    []payload.ValidationErrorDetail{{Location: "/id", Error: "got number, want string"}},
					},
				})
			},
		},
		{
			name:               "unsuccessful opengraph payload, node kinds validation error",
			payload:            `{"metadata":{"source_kind":"hellobase"},"graph":{"nodes":[{"id":"TESTNODE","kinds":["User", 1]}]}}`,
			expectedParsedData: payload.ParsedData{PayloadType: ingest.DataTypeOpenGraph, OpengraphData: payload.ParsedOpenGraphData{Metadata: ingest.OpengraphMetadata{SourceKind: "hellobase"}, NodesValidated: 1}},
			errValidationFunc: func(t *testing.T, report payload.ValidationReport, err error) {
				assert.ErrorIs(t, err, payload.ErrValidationErrors)

				assert.ElementsMatch(t, report.ValidationErrors, []payload.ValidationError{
					{
						Location:  "/graph/nodes[0]",
						RawObject: `{"id":"TESTNODE","kinds":["User", 1]}`,
						Errors:    []payload.ValidationErrorDetail{{Location: "/kinds/1", Error: "got number, want string"}},
					},
				})
			},
		},
		{
			name:               "unsuccessful opengraph payload, node properties validation error",
			payload:            `{"metadata":{"source_kind":"hellobase"},"graph":{"nodes":[{"id":"TESTNODE","kinds":["User"],"properties":{"items":{}}}]}}`,
			expectedParsedData: payload.ParsedData{PayloadType: ingest.DataTypeOpenGraph, OpengraphData: payload.ParsedOpenGraphData{Metadata: ingest.OpengraphMetadata{SourceKind: "hellobase"}, NodesValidated: 1}},
			errValidationFunc: func(t *testing.T, report payload.ValidationReport, err error) {
				assert.ErrorIs(t, err, payload.ErrValidationErrors)

				assert.ElementsMatch(t, report.ValidationErrors, []payload.ValidationError{
					{
						Location:  "/graph/nodes[0]",
						RawObject: `{"id":"TESTNODE","kinds":["User"],"properties":{"items":{}}}`,
						Errors:    []payload.ValidationErrorDetail{{Location: "/properties/items", Error: "invalid type"}},
					},
				})
			},
		},
		{
			name:               "unsuccessful opengraph payload, node multiple validation errors",
			payload:            `{"metadata":{"source_kind":"hellobase"},"graph":{"nodes":[{"id":1,"kinds":["User"],"properties":{"items":{}}}]}}`,
			expectedParsedData: payload.ParsedData{PayloadType: ingest.DataTypeOpenGraph, OpengraphData: payload.ParsedOpenGraphData{Metadata: ingest.OpengraphMetadata{SourceKind: "hellobase"}, NodesValidated: 1}},
			errValidationFunc: func(t *testing.T, report payload.ValidationReport, err error) {
				assert.ErrorIs(t, err, payload.ErrValidationErrors)

				require.Len(t, report.ValidationErrors, 1)
				require.Equal(t, "/graph/nodes[0]", report.ValidationErrors[0].Location)
				require.Equal(t, `{"id":1,"kinds":["User"],"properties":{"items":{}}}`, report.ValidationErrors[0].RawObject)
				assert.ElementsMatch(t, report.ValidationErrors[0].Errors, []payload.ValidationErrorDetail{{Location: "/id", Error: "got number, want string"}, {Location: "/properties/items", Error: "invalid type"}})
			},
		},
		{
			name: "unsuccessful opengraph payload, exceeds max validation errors",
			payload: `{"metadata":{"source_kind":"hellobase"},"graph":{"nodes":[{"id":"1","kinds":["A","A","A","A"]},` +
				`{"id":"1","kinds":["A","A","A","A"]},{"id":"1","kinds":["A","A","A","A"]},{"id":"1","kinds":["A","A","A","A"]},{"id":"1","kinds":["A","A","A","A"]},` +
				`{"id":"1","kinds":["A","A","A","A"]},{"id":"1","kinds":["A","A","A","A"]},{"id":"1","kinds":["A","A","A","A"]},{"id":"1","kinds":["A","A","A","A"]},` +
				`{"id":"1","kinds":["A","A","A","A"]},{"id":"1","kinds":["A","A","A","A"]},{"id":"1","kinds":["A","A","A","A"]},{"id":"1","kinds":["A","A","A","A"]},` +
				`{"id":"1","kinds":["A","A","A","A"]},{"id":"1","kinds":["A","A","A","A"]},{"id":"1","kinds":["A","A","A","A"]},{"id":"1","kinds":["A","A","A","A"]}]}}`,
			expectedParsedData: payload.ParsedData{PayloadType: ingest.DataTypeOpenGraph, OpengraphData: payload.ParsedOpenGraphData{Metadata: ingest.OpengraphMetadata{SourceKind: "hellobase"}, NodesValidated: 15}},
			errValidationFunc: func(t *testing.T, report payload.ValidationReport, err error) {
				assert.ErrorIs(t, err, payload.ErrMaxValidationErrors)

				assert.ElementsMatch(t, report.ValidationErrors, []payload.ValidationError{
					{Location: "/graph/nodes[0]", RawObject: `{"id":"1","kinds":["A","A","A","A"]}`, Errors: []payload.ValidationErrorDetail{{Location: "/kinds", Error: "maxItems: got 4, want 3"}}},
					{Location: "/graph/nodes[1]", RawObject: `{"id":"1","kinds":["A","A","A","A"]}`, Errors: []payload.ValidationErrorDetail{{Location: "/kinds", Error: "maxItems: got 4, want 3"}}},
					{Location: "/graph/nodes[2]", RawObject: `{"id":"1","kinds":["A","A","A","A"]}`, Errors: []payload.ValidationErrorDetail{{Location: "/kinds", Error: "maxItems: got 4, want 3"}}},
					{Location: "/graph/nodes[3]", RawObject: `{"id":"1","kinds":["A","A","A","A"]}`, Errors: []payload.ValidationErrorDetail{{Location: "/kinds", Error: "maxItems: got 4, want 3"}}},
					{Location: "/graph/nodes[4]", RawObject: `{"id":"1","kinds":["A","A","A","A"]}`, Errors: []payload.ValidationErrorDetail{{Location: "/kinds", Error: "maxItems: got 4, want 3"}}},
					{Location: "/graph/nodes[5]", RawObject: `{"id":"1","kinds":["A","A","A","A"]}`, Errors: []payload.ValidationErrorDetail{{Location: "/kinds", Error: "maxItems: got 4, want 3"}}},
					{Location: "/graph/nodes[6]", RawObject: `{"id":"1","kinds":["A","A","A","A"]}`, Errors: []payload.ValidationErrorDetail{{Location: "/kinds", Error: "maxItems: got 4, want 3"}}},
					{Location: "/graph/nodes[7]", RawObject: `{"id":"1","kinds":["A","A","A","A"]}`, Errors: []payload.ValidationErrorDetail{{Location: "/kinds", Error: "maxItems: got 4, want 3"}}},
					{Location: "/graph/nodes[8]", RawObject: `{"id":"1","kinds":["A","A","A","A"]}`, Errors: []payload.ValidationErrorDetail{{Location: "/kinds", Error: "maxItems: got 4, want 3"}}},
					{Location: "/graph/nodes[9]", RawObject: `{"id":"1","kinds":["A","A","A","A"]}`, Errors: []payload.ValidationErrorDetail{{Location: "/kinds", Error: "maxItems: got 4, want 3"}}},
					{Location: "/graph/nodes[10]", RawObject: `{"id":"1","kinds":["A","A","A","A"]}`, Errors: []payload.ValidationErrorDetail{{Location: "/kinds", Error: "maxItems: got 4, want 3"}}},
					{Location: "/graph/nodes[11]", RawObject: `{"id":"1","kinds":["A","A","A","A"]}`, Errors: []payload.ValidationErrorDetail{{Location: "/kinds", Error: "maxItems: got 4, want 3"}}},
					{Location: "/graph/nodes[12]", RawObject: `{"id":"1","kinds":["A","A","A","A"]}`, Errors: []payload.ValidationErrorDetail{{Location: "/kinds", Error: "maxItems: got 4, want 3"}}},
					{Location: "/graph/nodes[13]", RawObject: `{"id":"1","kinds":["A","A","A","A"]}`, Errors: []payload.ValidationErrorDetail{{Location: "/kinds", Error: "maxItems: got 4, want 3"}}},
					{Location: "/graph/nodes[14]", RawObject: `{"id":"1","kinds":["A","A","A","A"]}`, Errors: []payload.ValidationErrorDetail{{Location: "/kinds", Error: "maxItems: got 4, want 3"}}},
				})
			},
		},
		{
			name:               "successful opengraph payload with edge",
			payload:            `{"metadata":{"source_kind":"hellobase"},"graph":{"nodes":[],"edges":[{"start":{"value":"TESTNODE"},"end":{"value":"TESTNODE2"},"kind":"RELATED","properties":{"items":["hi"]}}]}}`,
			expectedParsedData: payload.ParsedData{PayloadType: ingest.DataTypeOpenGraph, OpengraphData: payload.ParsedOpenGraphData{Metadata: ingest.OpengraphMetadata{SourceKind: "hellobase"}, EdgesValidated: 1}},
			errValidationFunc: func(t *testing.T, report payload.ValidationReport, err error) {
				assert.Equal(t, emptyValidationReport, report)
				assert.NoError(t, err)
			},
		},
		{
			name:               "successful opengraph payload with edge property matching",
			payload:            `{"metadata":{"source_kind":"hellobase"},"graph":{"nodes":[],"edges":[{"start":{"match_by":"property","property_matchers":[{"key":"prop_1","operator":"equals","value":"ROHAN"}]},"end":{"match_by":"property","property_matchers":[{"key":"prop_1","operator":"equals","value":"WES"}]},"kind":"RELATED","properties":{"items":["hi"]}}]}}`,
			expectedParsedData: payload.ParsedData{PayloadType: ingest.DataTypeOpenGraph, OpengraphData: payload.ParsedOpenGraphData{Metadata: ingest.OpengraphMetadata{SourceKind: "hellobase"}, EdgesValidated: 1}},
			errValidationFunc: func(t *testing.T, report payload.ValidationReport, err error) {
				assert.Equal(t, emptyValidationReport, report)
				assert.NoError(t, err)
			},
		},
		{
			name:               "unsuccessful opengraph payload, edge properties validation error",
			payload:            `{"metadata":{"source_kind":"hellobase"},"graph":{"nodes":[],"edges":[{"start":{"value":"TESTNODE"},"end":{"value":"TESTNODE2"},"kind":"RELATED","properties":{"items":{}}}]}}`,
			expectedParsedData: payload.ParsedData{PayloadType: ingest.DataTypeOpenGraph, OpengraphData: payload.ParsedOpenGraphData{Metadata: ingest.OpengraphMetadata{SourceKind: "hellobase"}, EdgesValidated: 1}},
			errValidationFunc: func(t *testing.T, report payload.ValidationReport, err error) {
				assert.ErrorIs(t, err, payload.ErrValidationErrors)

				assert.ElementsMatch(t, report.ValidationErrors, []payload.ValidationError{
					{
						Location:  "/graph/edges[0]",
						RawObject: `{"start":{"value":"TESTNODE"},"end":{"value":"TESTNODE2"},"kind":"RELATED","properties":{"items":{}}}`,
						Errors:    []payload.ValidationErrorDetail{{Location: "/properties/items", Error: "invalid type"}},
					},
				})
			},
		},
		{
			name:               "unsuccessful opengraph payload, edge id validation error",
			payload:            `{"metadata":{"source_kind":"hellobase"},"graph":{"nodes":[],"edges":[{"start":{"value":1},"end":{"value":"TESTNODE2"},"kind":"RELATED","properties":{"items":["hi"]}}]}}`,
			expectedParsedData: payload.ParsedData{PayloadType: ingest.DataTypeOpenGraph, OpengraphData: payload.ParsedOpenGraphData{Metadata: ingest.OpengraphMetadata{SourceKind: "hellobase"}, EdgesValidated: 1}},
			errValidationFunc: func(t *testing.T, report payload.ValidationReport, err error) {
				assert.ErrorIs(t, err, payload.ErrValidationErrors)

				assert.ElementsMatch(t, report.ValidationErrors, []payload.ValidationError{
					{
						Location:  "/graph/edges[0]",
						RawObject: `{"start":{"value":1},"end":{"value":"TESTNODE2"},"kind":"RELATED","properties":{"items":["hi"]}}`,
						Errors:    []payload.ValidationErrorDetail{{Location: "/start/value", Error: "got number, want string"}},
					},
				})
			},
		},
		{
			name:               "unsuccessful opengraph payload, invalid edge property matching",
			payload:            `{"metadata":{"source_kind":"hellobase"},"graph":{"nodes":[],"edges":[{"start":{"match_by":"property","property_matchers":{"key":"prop_1","operator":"equals","value":"ROHAN"}},"end":{"match_by":"property","property_matchers":[{"key":"prop_1","operator":"equals","value":"WES"}]},"kind":"RELATED","properties":{"items":["hi"]}}]}}`,
			expectedParsedData: payload.ParsedData{PayloadType: ingest.DataTypeOpenGraph, OpengraphData: payload.ParsedOpenGraphData{Metadata: ingest.OpengraphMetadata{SourceKind: "hellobase"}, EdgesValidated: 1}},
			errValidationFunc: func(t *testing.T, report payload.ValidationReport, err error) {
				assert.ErrorIs(t, err, payload.ErrValidationErrors)

				assert.ElementsMatch(t, report.ValidationErrors, []payload.ValidationError{
					{
						Location:  "/graph/edges[0]",
						RawObject: `{"start":{"match_by":"property","property_matchers":{"key":"prop_1","operator":"equals","value":"ROHAN"}},"end":{"match_by":"property","property_matchers":[{"key":"prop_1","operator":"equals","value":"WES"}]},"kind":"RELATED","properties":{"items":["hi"]}}`,
						Errors:    []payload.ValidationErrorDetail{{Location: "/start/property_matchers", Error: "got object, want array"}},
					},
				})
			},
		},
		{
			name:               "unsuccessful opengraph metadata",
			payload:            `{"metadata":{"source_kind":1},"graph":{"nodes":[]}}`,
			expectedParsedData: payload.ParsedData{},
			errValidationFunc: func(t *testing.T, report payload.ValidationReport, err error) {
				assert.ErrorIs(t, err, payload.ErrOpengraphMetadataValidation)

				assert.ElementsMatch(t, report.CriticalErrors, []payload.CriticalError{{Message: "opengraph metadata failed validation", Error: payload.ErrOpengraphMetadataValidation}})
			},
		},
		{
			name:               "unsuccessful opengraph no child tags",
			payload:            `{"graph":{}}`,
			expectedParsedData: payload.ParsedData{PayloadType: ingest.DataTypeOpenGraph},
			errValidationFunc: func(t *testing.T, report payload.ValidationReport, err error) {
				assert.ErrorIs(t, err, payload.ErrInvalidFileConfiguration)

				assert.ElementsMatch(t, report.CriticalErrors, []payload.CriticalError{{Message: "graph tag requires child nodes or edges tag", Error: payload.ErrInvalidFileConfiguration}})
			},
		},
		{
			name:               "unsuccessful opengraph metadata, invalid field",
			payload:            `{"metadata":{"random field":"hello"},"graph":{"nodes":[]}}`,
			expectedParsedData: payload.ParsedData{},
			errValidationFunc: func(t *testing.T, report payload.ValidationReport, err error) {
				assert.ErrorIs(t, err, payload.ErrOpengraphMetadataValidation)

				assert.ElementsMatch(t, report.CriticalErrors, []payload.CriticalError{{Message: "opengraph metadata failed validation", Error: payload.ErrOpengraphMetadataValidation}})
			},
		},
		// Original payload tests
		{
			name:               "successful original payload",
			payload:            `{"meta":{"methods": 0,"type":"sessions","count": 0,"version": 5},"data":[]}`,
			expectedParsedData: payload.ParsedData{PayloadType: ingest.DataTypeSession, LegacyMetadata: ingest.OriginalMetadata{Type: ingest.DataTypeSession, Methods: 0, Version: 5}},
			errValidationFunc: func(t *testing.T, report payload.ValidationReport, err error) {
				assert.Equal(t, emptyValidationReport, report)
				assert.NoError(t, err)
			},
		},
		{
			name:               "unsuccessful original payload, no data tag",
			payload:            `{"meta":{"methods": 0,"type":"sessions","count": 0,"version":5}}`,
			expectedParsedData: payload.ParsedData{PayloadType: ingest.DataTypeSession, LegacyMetadata: ingest.OriginalMetadata{Type: ingest.DataTypeSession, Methods: 0, Version: 5}},
			errValidationFunc: func(t *testing.T, report payload.ValidationReport, err error) {
				assert.ErrorIs(t, err, payload.ErrInvalidFileConfiguration)

				assert.ElementsMatch(t, report.CriticalErrors, []payload.CriticalError{{Message: "no data tag found to match original metadata tag", Error: payload.ErrInvalidFileConfiguration}})
			},
		},
		{
			name:               "unsuccessful original payload, no meta tag",
			payload:            `{"data":[]}`,
			expectedParsedData: payload.ParsedData{},
			errValidationFunc: func(t *testing.T, report payload.ValidationReport, err error) {
				assert.ErrorIs(t, err, payload.ErrInvalidFileConfiguration)

				assert.ElementsMatch(t, report.CriticalErrors, []payload.CriticalError{{Message: "no meta tag found to match original data tag", Error: payload.ErrInvalidFileConfiguration}})
			},
		},
		{
			name:               "unsuccessful original payload, duplicate meta tag",
			payload:            `{"meta":{"methods":0,"type":"sessions","count":0,"version":5},"meta":0,"data":[]}`,
			expectedParsedData: payload.ParsedData{PayloadType: ingest.DataTypeSession, LegacyMetadata: ingest.OriginalMetadata{Type: ingest.DataTypeSession, Methods: 0, Version: 5}},
			errValidationFunc: func(t *testing.T, report payload.ValidationReport, err error) {
				assert.ErrorIs(t, err, payload.ErrInvalidFileConfiguration)

				assert.ElementsMatch(t, report.CriticalErrors, []payload.CriticalError{{Message: "duplicate top level meta tag found", Error: payload.ErrInvalidFileConfiguration}})
			},
		},
		{
			name:               "unsuccessful original payload, invalid meta",
			payload:            `{"data":[],"meta":0}`,
			expectedParsedData: payload.ParsedData{},
			errValidationFunc: func(t *testing.T, report payload.ValidationReport, err error) {
				require.Len(t, report.CriticalErrors, 1)
				var (
					criticalError = report.CriticalErrors[0]
					unmarshalErr  = &json.UnmarshalTypeError{}
				)

				assert.Equal(t, "failed to decode original metadata", criticalError.Message)
				assert.ErrorAs(t, criticalError.Error, &unmarshalErr)
				assert.ErrorAs(t, err, &unmarshalErr)
			},
		},
		{
			name:               "swapped order",
			payload:            `{"data":[],"meta":{"methods":0,"type":"sessions","count":0,"version":5}}`,
			expectedParsedData: payload.ParsedData{PayloadType: ingest.DataTypeSession, LegacyMetadata: ingest.OriginalMetadata{Type: ingest.DataTypeSession, Methods: 0, Version: 5}},
			errValidationFunc: func(t *testing.T, report payload.ValidationReport, err error) {
				assert.Equal(t, emptyValidationReport, report)
				assert.NoError(t, err)
			},
		},
		{
			name:               "unsuccessful original payload, invalid type",
			payload:            `{"data":[],"meta":{"methods":0,"type":"invalid","count":0,"version":5}}`,
			expectedParsedData: payload.ParsedData{},
			errValidationFunc: func(t *testing.T, report payload.ValidationReport, err error) {
				assert.ErrorIs(t, err, payload.ErrInvalidDataType)

				assert.ElementsMatch(t, report.CriticalErrors, []payload.CriticalError{{Message: "invalid original metadata data type", Error: payload.ErrInvalidDataType}})
			},
		},
		// Invalid payload tests
		{
			name:               "unsuccessful payload, no valid tags",
			payload:            `{}`,
			expectedParsedData: payload.ParsedData{},
			errValidationFunc: func(t *testing.T, report payload.ValidationReport, err error) {
				assert.ErrorIs(t, err, payload.ErrInvalidFileConfiguration)

				assert.ElementsMatch(t, report.CriticalErrors, []payload.CriticalError{{Message: "no tags found", Error: payload.ErrInvalidFileConfiguration}})
			},
		},
		{
			name:               "enforce mutual exclusivity",
			payload:            `{"data":[],"graph":{}}`,
			expectedParsedData: payload.ParsedData{},
			errValidationFunc: func(t *testing.T, report payload.ValidationReport, err error) {
				assert.ErrorIs(t, err, payload.ErrInvalidFileConfiguration)

				assert.ElementsMatch(t, report.CriticalErrors, []payload.CriticalError{{Message: "cannot have both original data tag and opengraph graph tag", Error: payload.ErrInvalidFileConfiguration}})
			},
		},
		{
			name:               "unsuccessful payload, unrecognized top level tag",
			payload:            `{"graph":{"nodes":[]},"pants":{}}`,
			expectedParsedData: payload.ParsedData{PayloadType: ingest.DataTypeOpenGraph},
			errValidationFunc: func(t *testing.T, report payload.ValidationReport, err error) {
				assert.ErrorIs(t, err, payload.ErrInvalidFileConfiguration)

				assert.ElementsMatch(t, report.CriticalErrors, []payload.CriticalError{{Message: "unrecognized top level tag: pants", Error: payload.ErrInvalidFileConfiguration}})
			},
		},
		{
			name:               "unsuccessful payload, trailing data after object",
			payload:            `{"graph":{"nodes":[]}}{}`,
			expectedParsedData: payload.ParsedData{PayloadType: ingest.DataTypeOpenGraph},
			errValidationFunc: func(t *testing.T, report payload.ValidationReport, err error) {
				assert.ErrorContains(t, err, "expected EOF, instead got token: {")
				require.Len(t, report.CriticalErrors, 1)
				assert.Equal(t, "expected to hit the end of the file", report.CriticalErrors[0].Message)
				assert.ErrorContains(t, report.CriticalErrors[0].Error, "expected EOF, instead got token: {")
			},
		},
	}

	schema, err := payload.LoadSchema()
	require.NoError(t, err)

	for _, assertion := range assertions {
		t.Run(assertion.name, func(t *testing.T) {
			v := payload.NewValidator(strings.NewReader(assertion.payload), schema)

			parsedData, validationReport, err := v.ParseAndValidate()
			assert.Equal(t, assertion.expectedParsedData, parsedData)
			assertion.errValidationFunc(t, validationReport, err)
		})
	}
}

type parseMetadataAssertion struct {
	name               string
	payload            string
	expectedParsedData payload.ParsedData
	errValidationFunc  func(t *testing.T, err error)
}

func Test_ParseMetadata(t *testing.T) {
	assertions := []parseMetadataAssertion{
		{
			name:    "legacy metadata",
			payload: `{"meta":{"methods":0,"type":"sessions","count":0,"version":5},"data":[]}`,
			expectedParsedData: payload.ParsedData{
				PayloadType:    ingest.DataTypeSession,
				LegacyMetadata: ingest.OriginalMetadata{Type: ingest.DataTypeSession, Methods: 0, Version: 5},
			},
			errValidationFunc: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:    "legacy metadata after data",
			payload: `{"data":[],"meta":{"methods":0,"type":"sessions","count":0,"version":5}}`,
			expectedParsedData: payload.ParsedData{
				PayloadType:    ingest.DataTypeSession,
				LegacyMetadata: ingest.OriginalMetadata{Type: ingest.DataTypeSession, Methods: 0, Version: 5},
			},
			errValidationFunc: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:    "opengraph metadata",
			payload: `{"metadata":{"source_kind":"hellobase"},"graph":{"nodes":[]}}`,
			expectedParsedData: payload.ParsedData{
				PayloadType: ingest.DataTypeOpenGraph,
				OpengraphData: payload.ParsedOpenGraphData{
					Metadata: ingest.OpengraphMetadata{SourceKind: "hellobase"},
				},
			},
			errValidationFunc: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:    "opengraph graph only",
			payload: `{"graph":{"nodes":[]}}`,
			expectedParsedData: payload.ParsedData{
				PayloadType: ingest.DataTypeOpenGraph,
			},
			errValidationFunc: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:    "opengraph metadata after graph",
			payload: `{"graph":{"nodes":[]},"metadata":{"source_kind":"hellobase"}}`,
			expectedParsedData: payload.ParsedData{
				PayloadType: ingest.DataTypeOpenGraph,
				OpengraphData: payload.ParsedOpenGraphData{
					Metadata: ingest.OpengraphMetadata{SourceKind: "hellobase"},
				},
			},
			errValidationFunc: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:               "invalid top level json",
			payload:            `[]`,
			expectedParsedData: payload.ParsedData{},
			errValidationFunc: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "expected open bracket")
			},
		},
	}

	schema, err := payload.LoadSchema()
	require.NoError(t, err)

	for _, assertion := range assertions {
		t.Run(assertion.name, func(t *testing.T) {
			v := payload.NewValidator(strings.NewReader(assertion.payload), schema)

			parsedData, err := v.ParseMetadata()
			assert.Equal(t, assertion.expectedParsedData, parsedData)
			assertion.errValidationFunc(t, err)
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	validationErr := payload.ValidationError{
		Location: "/graph/nodes[0]",
		Errors: []payload.ValidationErrorDetail{
			{Location: "/id", Error: "got number, want string"},
			{Location: "/properties/items", Error: "invalid type"},
		},
	}

	assert.Equal(t, "validation error at /graph/nodes[0]: /id: got number, want string; /properties/items: invalid type", validationErr.Error())
}

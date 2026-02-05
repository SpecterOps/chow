package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/require"
)

const nodePayload = `{
	"id": "TESTNODE",
	"kinds": ["User", "User", "User", "User"],
	"properties": {
		"prop1": "testing",
		"prop2": 1,
		"prop3": "3",
		"prop4": ["hello", ":D"],
		"test": {},
		"test2": [],
		"test3": ["hi", 0, true],
		"test4": [0, 0, 0],
		"test5": [true, false, false]
	}
}`

func TestExtractJsonSchemaErrors(t *testing.T) {
	schema, err := LoadIngestSchema()
	require.NoError(t, err)

	var item map[string]any
	err = json.Unmarshal([]byte(nodePayload), &item)
	require.NoError(t, err)

	err = schema.NodeSchema.Validate(item)
	var schemaErr *jsonschema.ValidationError
	ok := errors.As(err, &schemaErr)
	require.True(t, ok)
	errors := extractJsonSchemaErrors(schemaErr)
	fmt.Println(errors)
	require.NotEmpty(t, errors)
}

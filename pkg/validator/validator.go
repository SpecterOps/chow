package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/SpecterOps/chow/pkg/ingest"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const (
	DelimOpenBracket        = json.Delim('{')
	DelimCloseBracket       = json.Delim('}')
	DelimOpenSquareBracket  = json.Delim('[')
	DelimCloseSquareBracket = json.Delim(']')
)

type CriticalError struct {
	Message string
}

type ValidationError struct {
	Location  string
	RawObject string
	Errors    []ValidationErrorDetail
}

type ValidationErrorDetail struct {
	Location string
	Error    string
}

type Validator struct {
	decoder *json.Decoder
	depth   int

	schema    IngestSchema
	readToEnd bool

	criticalErrors   []CriticalError
	validationErrors []ValidationError
}

type ValidationReport struct {
	CriticalErrors   []CriticalError
	ValidationErrors []ValidationError
}

func NewValidator(decoder *json.Decoder, schema IngestSchema) Validator {
	return Validator{
		decoder: decoder,
		depth:   0,

		schema: schema,

		criticalErrors:   make([]CriticalError, 0),
		validationErrors: make([]ValidationError, 0),
	}
}

// External validation function
func (v *Validator) Validate() (ValidationReport, error) {
	if err := v.enterObject(); err != nil {
		return ValidationReport{}, err
	}

	if err := v.validationLoop(); err != nil {
		return ValidationReport{}, err
	}

	// need to make sure we hit EOF

	return ValidationReport{
		CriticalErrors:   v.criticalErrors,
		ValidationErrors: v.validationErrors,
	}, nil
}

// Validation Loop functions
func (v *Validator) validationLoop() error {
	for {
		if tag, exitedBlock, err := v.nextTagAtDepth(1); err != nil {
			return err
		} else if exitedBlock {
			return nil
		} else {
			switch tag {
			case "meta":
				_, err := v.parseLegacyMetadata()
				return err
			case "data":
				return v.parseData()
			case "metadata":
				_, err := v.parseOpenGraphMetadata()
				return err
			case "graph":
				return v.parseGraph()
			}
		}
	}
}

func (v *Validator) parseLegacyMetadata() (ingest.LegacyMetadata, error) {
	var legacyMetadata ingest.LegacyMetadata
	if err := v.decoder.Decode(&legacyMetadata); err != nil {
		return legacyMetadata, err
	}
	return ingest.LegacyMetadata{}, nil
}

func (v *Validator) parseOpenGraphMetadata() (ingest.OpengraphMetadata, error) {
	var opengraphMetadata ingest.OpengraphMetadata
	if err := v.decoder.Decode(&opengraphMetadata); err != nil {
		return opengraphMetadata, err
	}
	return ingest.OpengraphMetadata{}, nil
}

func (v *Validator) parseData() error {
	if err := v.enterArray(); err != nil {
		return err
	}

	return nil
}

func (v *Validator) parseGraph() error {
	if err := v.enterObject(); err != nil {
		return err
	}

	for {
		if tag, exitedBlock, err := v.nextTagAtDepth(2); err != nil {
			return err
		} else if exitedBlock {
			return nil
		} else {
			switch tag {
			case "nodes":
				return v.parseGraphNodes()
			case "edges":
				return v.parseGraphEdges()
			}
		}
	}
}

func (v *Validator) parseGraphNodes() error {
	if err := v.enterArray(); err != nil {
		return err
	}

	index := 0
	for v.decoder.More() {
		var rawItem json.RawMessage

		err := v.decoder.Decode(&rawItem)
		if err != nil {
			return err
		}

		var item map[string]any

		err = json.Unmarshal(rawItem, &item)
		if err != nil {
			return err
		}

		err = v.schema.NodeSchema.Validate(item)
		if err != nil {
			location := fmt.Sprintf("/graph/nodes[%d]", index)

			var schemaErr *jsonschema.ValidationError
			if ok := errors.As(err, &schemaErr); !ok {
				continue
			}

			v.validationErrors = append(v.validationErrors, ValidationError{
				Location:  location,
				RawObject: string(rawItem),
				Errors:    extractJsonSchemaErrors(schemaErr),
			})
		}
	}

	return nil
}

func (v *Validator) parseGraphEdges() error {
	if err := v.enterArray(); err != nil {
		return err
	}

	index := 0
	for v.decoder.More() {
		var rawItem json.RawMessage

		err := v.decoder.Decode(&rawItem)
		if err != nil {
			return err
		}

		var item map[string]any

		err = json.Unmarshal(rawItem, &item)
		if err != nil {
			return err
		}

		err = v.schema.EdgeSchema.Validate(item)
		if err != nil {
			location := fmt.Sprintf("/graph/edges[%d]", index)

			var schemaErr *jsonschema.ValidationError
			if ok := errors.As(err, &schemaErr); !ok {
				continue
			}

			v.validationErrors = append(v.validationErrors, ValidationError{
				Location:  location,
				RawObject: string(rawItem),
				Errors:    extractJsonSchemaErrors(schemaErr),
			})
		}
	}

	return nil
}

func extractJsonSchemaErrors(ve *jsonschema.ValidationError) []ValidationErrorDetail {
	errors := make(map[string]string, 0)

	for _, cause := range ve.Causes {
		if cause.BasicOutput() == nil {
			continue
		}

		output := cause.BasicOutput()

		if output.Errors != nil {
			for _, e := range output.Errors {
				if e.Error == nil {
					continue
				}

				locSplit := strings.Split(e.InstanceLocation, "/")
				if len(locSplit) < 3 {
					continue
				}

				if locSplit[1] == "properties" {
					newLocation := fmt.Sprintf("/%s/%s", locSplit[1], locSplit[2])
					if _, ok := errors[newLocation]; !ok {
						errors[newLocation] = e.Error.String()
					}
				} else {
					if _, ok := errors[e.InstanceLocation]; !ok {
						errors[e.InstanceLocation] = e.Error.String()
					}
				}
			}
		} else if output.Error != nil {
			errors[output.InstanceLocation] = output.Error.String()
		}
	}

	errorDetails := make([]ValidationErrorDetail, 0)
	for loc, err := range errors {
		errorDetails = append(errorDetails, ValidationErrorDetail{
			Location: loc,
			Error:    err,
		})
	}

	return errorDetails
}

// Scanner functions

func (v *Validator) enterObject() error {
	t, err := v.nextToken()
	if err != nil {
		return err
	}

	if delim, ok := t.(json.Delim); !ok || delim != DelimOpenBracket {
		return fmt.Errorf("expected open bracket")
	}

	return nil
}

func (v *Validator) exitObject() error {
	t, err := v.nextToken()
	if err != nil {
		return err
	}

	if delim, ok := t.(json.Delim); !ok || delim != DelimCloseBracket {
		return fmt.Errorf("expected closing bracket")
	}

	return nil
}

func (v *Validator) enterArray() error {
	t, err := v.nextToken()
	if err != nil {
		return err
	}

	if delim, ok := t.(json.Delim); !ok || delim != DelimOpenSquareBracket {
		return fmt.Errorf("expected open bracket")
	}

	return nil
}

func (v *Validator) exitArray() error {
	t, err := v.nextToken()
	if err != nil {
		return err
	}

	if delim, ok := t.(json.Delim); !ok || delim != DelimCloseSquareBracket {
		return fmt.Errorf("expected open bracket")
	}

	return nil
}

func (v *Validator) nextTagAtDepth(depth int) (string, bool, error) {
	for {
		t, err := v.nextToken()
		if err != nil {
			return "", false, err
		}

		if v.depth < depth {
			return "", true, nil
		}

		tag, ok := t.(string)
		if !ok {
			continue
		}

		if v.depth == depth {
			return tag, false, nil
		}
	}
}

func (v *Validator) nextToken() (json.Token, error) {
	tok, err := v.decoder.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := tok.(json.Delim); ok {
		if d == DelimOpenBracket || d == DelimOpenSquareBracket {
			v.depth++
		} else {
			v.depth--
		}
	}
	return tok, nil
}

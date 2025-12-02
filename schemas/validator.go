package schemas

import (
	_ "embed"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

//go:embed instruction-schema.json
var schemaBytes []byte

var schemaValidator *gojsonschema.Schema

func init() {
	loader := gojsonschema.NewBytesLoader(schemaBytes)
	var err error
	schemaValidator, err = gojsonschema.NewSchema(loader)
	if err != nil {
		panic(fmt.Sprintf("failed to load schema: %v", err))
	}
}

// ValidateInstruction validates JSON instruction data against the schema
func ValidateInstruction(data []byte) error {
	documentLoader := gojsonschema.NewBytesLoader(data)
	result, err := schemaValidator.Validate(documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}

	if !result.Valid() {
		var errorMsg string
		for _, desc := range result.Errors() {
			if errorMsg != "" {
				errorMsg += "; "
			}
			errorMsg += desc.String()
		}
		return fmt.Errorf("schema validation failed: %s", errorMsg)
	}

	return nil
}

// ValidateInstructionStruct validates a SwapInstruction struct against the schema
func ValidateInstructionStruct(instruction *SwapInstruction) error {
	data, err := instruction.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize instruction: %w", err)
	}

	return ValidateInstruction(data)
}

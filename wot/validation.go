package wot

import (
	"fmt"
	"io/ioutil"

	"github.com/xeipuuv/gojsonschema"
)

type jsonSchema = *gojsonschema.Schema

var loadedJSONSchemas []jsonSchema

// ReadJSONSchema reads the a JSONSchema from a file
func readJSONSchema(path string) (jsonSchema, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %s", err)
	}

	schema, err := gojsonschema.NewSchema(gojsonschema.NewBytesLoader(file))
	if err != nil {
		return nil, fmt.Errorf("error loading schema: %s", err)
	}
	return schema, nil
}

// LoadJSONSchemas loads one or more JSON Schemas into memory
func LoadJSONSchemas(paths []string) error {
	if len(loadedJSONSchemas) != 0 {
		panic("Unexpected re-loading of JSON Schemas.")
	}
	var schemas []jsonSchema
	for _, path := range paths {
		schema, err := readJSONSchema(path)
		if err != nil {
			return err
		}
		schemas = append(schemas, schema)
	}
	loadedJSONSchemas = schemas
	return nil
}

// LoadedJSONSchemas checks whether any JSON Schema has been loaded into memory
func LoadedJSONSchemas() bool {
	return len(loadedJSONSchemas) > 0
}

func validateAgainstSchema(td *map[string]interface{}, schema jsonSchema) ([]ValidationError, error) {
	result, err := schema.Validate(gojsonschema.NewGoLoader(td))
	if err != nil {
		return nil, err
	}

	if !result.Valid() {
		var issues []ValidationError
		for _, re := range result.Errors() {
			issues = append(issues, ValidationError{Field: re.Field(), Descr: re.Description()})
		}
		return issues, nil
	}

	return nil, nil
}

func validateAgainstSchemas(td *map[string]interface{}, schemas ...jsonSchema) ([]ValidationError, error) {
	var validationErrors []ValidationError
	for _, schema := range schemas {
		result, err := validateAgainstSchema(td, schema)
		if err != nil {
			return nil, err
		}
		validationErrors = append(validationErrors, result...)
	}

	return validationErrors, nil
}

// ValidateTD performs input validation using one or more pre-loaded JSON Schemas
// If no schema has been pre-loaded, the function returns as if there are no validation errors
func ValidateTD(td *map[string]interface{}) ([]ValidationError, error) {
	return validateAgainstSchemas(td, loadedJSONSchemas...)
}

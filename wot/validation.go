package wot

import (
	"fmt"
	"io/ioutil"

	"github.com/xeipuuv/gojsonschema"
)

// Deprecated
var schema *gojsonschema.Schema

// Deprecated
// LoadSchema loads the schema into the package
func LoadSchema(path string) error {
	if schema != nil {
		// already loaded
		return nil
	}

	file, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading file: %s", err)
	}

	schema, err = gojsonschema.NewSchema(gojsonschema.NewBytesLoader(file))
	if err != nil {
		return fmt.Errorf("error loading schema: %s", err)
	}
	return nil
}

// Deprecated
// ValidateMap validates the input against the loaded WoT Thing Description schema
func ValidateMap(td *map[string]interface{}) ([]ValidationError, error) {
	if schema == nil {
		return nil, fmt.Errorf("WoT Thing Description schema is not loaded")
	}

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

type jsonSchema = *gojsonschema.Schema

// LoadJSONSchema loads the a JSONSchema from a path
func LoadJSONSchema(path string) (jsonSchema, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %s", err)
	}

	schema, err = gojsonschema.NewSchema(gojsonschema.NewBytesLoader(file))
	if err != nil {
		return nil, fmt.Errorf("error loading schema: %s", err)
	}
	return schema, nil
}

// TODO: load schemas into memory (only once) instead of returning
// LoadJSONSchemas loads one or more JSON Schemas
func LoadJSONSchemas(paths []string) ([]jsonSchema, error) {
	var schemas []jsonSchema
	for _, path := range paths {
		schema, err := LoadJSONSchema(path)
		if err != nil {
			return nil, err
		}
		schemas = append(schemas, schema)
	}
	return schemas, nil
}

func validate(td *map[string]interface{}, schema jsonSchema) ([]ValidationError, error) {
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

// ValidateTD performs input validation using one or more JSON schemas
func ValidateTD(td *map[string]interface{}, schemas ...jsonSchema) ([]ValidationError, error) {
	var validationErrors []ValidationError
	for _, schema := range schemas {
		result, err := validate(td, schema)
		if err != nil {
			return nil, err
		}
		validationErrors = append(validationErrors, result...)
	}

	return validationErrors, nil
}

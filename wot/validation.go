package wot

import (
	"fmt"
	"io/ioutil"

	"github.com/xeipuuv/gojsonschema"
)

var schema *gojsonschema.Schema

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

func validate(td *map[string]interface{}, schema *gojsonschema.Schema) ([]ValidationError, error) {
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

func ValidateTD(td *map[string]interface{}, schemas ...*gojsonschema.Schema) ([]ValidationError, error) {
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

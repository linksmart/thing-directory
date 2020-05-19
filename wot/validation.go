package wot

import (
	"fmt"
	"io/ioutil"
	"strings"

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
func ValidateMap(td *map[string]interface{}) error {
	if schema == nil {
		return fmt.Errorf("WoT Thing Description schema is not loaded")
	}

	result, err := schema.Validate(gojsonschema.NewGoLoader(td))
	if err != nil {
		return err
	}

	if !result.Valid() {
		var errors []string
		for _, desc := range result.Errors() {
			errors = append(errors, desc.String())
		}
		return &ValidationError{errors}
	}
	return nil
}

type ValidationError struct {
	Errors []string
}

func (e ValidationError) Error() string {
	return strings.Join(e.Errors, ", ")
}

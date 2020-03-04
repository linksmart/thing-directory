package wot

import (
	"fmt"
	"strings"

	"github.com/xeipuuv/gojsonschema"
)

var schema *gojsonschema.Schema

func ValidateAgainstWoTSchema(td *map[string]interface{}) error {
	if schema == nil {
		// load schema into memory on first validation call
		var err error
		schema, err = gojsonschema.NewSchema(gojsonschema.NewStringLoader(jsonSchema))
		if err != nil {
			return fmt.Errorf("error loading WoT Schema: %s", err)
		}
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
		return fmt.Errorf("invalid Thing Description: %s", strings.Join(errors, ", "))
	}
	return nil
}

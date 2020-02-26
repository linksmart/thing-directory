package wot

import (
	"fmt"
	"strings"

	"github.com/xeipuuv/gojsonschema"
)

func ValidateAgainstWoTSchema(td *ThingDescription) error {

	result, err := gojsonschema.Validate(gojsonschema.NewStringLoader(WoTSchema), gojsonschema.NewGoLoader(td))
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

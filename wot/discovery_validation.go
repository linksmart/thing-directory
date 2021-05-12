package wot

import (
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

const DiscoverySchema = `
{
    "type":"object",
    "properties":{
        "registration":{
            "type":"object",
            "properties":{
                "created":{
                    "type":"string",
                    "format":"date-time"
                },
                "expires":{
                    "type":"string",
                    "format":"date-time"
                },
                "retrieved":{
                    "type":"string",
                    "format":"date-time"
                },
                "modified":{
                    "type":"string",
                    "format":"date-time"
                },
                "ttl":{
                    "type":"number"
                }
            }
        }
    }
}
`

func ValidateDiscoveryExtensions(td *map[string]interface{}) ([]ValidationError, error) {
	schema, err := gojsonschema.NewSchema(gojsonschema.NewStringLoader(DiscoverySchema))
	if err != nil {
		return nil, fmt.Errorf("error loading schema: %s", err)
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

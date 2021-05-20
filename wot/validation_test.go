package wot

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/xeipuuv/gojsonschema"
)

const (
	envTestSchemaPath = "TEST_SCHEMA_PATH"
	defaultSchemaPath = "../wot/wot_td_schema.json"
)

func TestLoadSchemas(t *testing.T) {
	if !LoadedJSONSchemas() {
		path := os.Getenv(envTestSchemaPath)
		if path == "" {
			path = defaultSchemaPath
		}
		err := LoadJSONSchemas([]string{path})
		if err != nil {
			t.Fatalf("error loading WoT Thing Description schema: %s", err)
		}
	}
	if len(loadedJSONSchemas) == 0 {
		t.Fatalf("JSON Schema was not loaded into memory")
	}
}

func TestValidateAgainstSchema(t *testing.T) {
	path := os.Getenv(envTestSchemaPath)
	if path == "" {
		path = defaultSchemaPath
	}

	// load the schema
	file, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("error reading file: %s", err)
	}
	schema, err := gojsonschema.NewSchema(gojsonschema.NewBytesLoader(file))
	if err != nil {
		t.Fatalf("error loading schema: %s", err)
	}

	t.Run("non-URI ID", func(t *testing.T) {
		var td = map[string]any{
			"@context": "https://www.w3.org/2019/wot/td/v1",
			"id":       "not-a-uri",
			"title":    "example thing",
			"security": []string{"basic_sc"},
			"securityDefinitions": map[string]any{
				"basic_sc": map[string]string{
					"in":     "header",
					"scheme": "basic",
				},
			},
		}
		results, err := validateAgainstSchema(&td, schema)
		if err != nil {
			t.Fatalf("internal validation error: %s", err)
		}
		if len(results) == 0 {
			t.Fatalf("Didn't return error on non-URI ID: %s", td["id"])
		}
	})

	t.Run("missing mandatory title", func(t *testing.T) {
		var td = map[string]any{
			"@context": "https://www.w3.org/2019/wot/td/v1",
			"id":       "not-a-uri",
			//"title":    "example thing",
			"security": []string{"basic_sc"},
			"securityDefinitions": map[string]any{
				"basic_sc": map[string]string{
					"in":     "header",
					"scheme": "basic",
				},
			},
		}
		results, err := validateAgainstSchema(&td, schema)
		if err != nil {
			t.Fatalf("internal validation error: %s", err)
		}
		if len(results) == 0 {
			t.Fatalf("Didn't return error on missing mandatory title.")
		}
	})

	// TODO test discovery validations
	//t.Run("non-float TTL", func(t *testing.T) {
	//	var td = map[string]any{
	//		"@context": "https://www.w3.org/2019/wot/td/v1",
	//		"id":       "urn:example:test/thing1",
	//		"title":    "example thing",
	//		"security": []string{"basic_sc"},
	//		"securityDefinitions": map[string]any{
	//			"basic_sc": map[string]string{
	//				"in":     "header",
	//				"scheme": "basic",
	//			},
	//		},
	//		"registration": map[string]any{
	//			"ttl": "60",
	//		},
	//	}
	//	results, err := validateAgainstSchema(&td, schema)
	//	if err != nil {
	//		t.Fatalf("internal validation error: %s", err)
	//	}
	//	if len(results) == 0 {
	//		t.Fatalf("Didn't return error on string TTL.")
	//	}
	//})
}

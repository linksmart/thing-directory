package wot

import (
	"os"
	"testing"
)

const (
	envTestSchemaPath = "TEST_SCHEMA_PATH"
	defaultSchemaPath = "../wot/wot_td_schema.json"
)

func TestLoadSchema(t *testing.T) {
	path := os.Getenv(envTestSchemaPath)
	if path == "" {
		path = defaultSchemaPath
	}
	err := LoadSchema(path)
	if err != nil {
		t.Fatalf("error loading WoT Thing Description schema: %s", err)
	}
}

func TestValidateMap(t *testing.T) {
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
		err := ValidateMap(&td)
		if err == nil {
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
		err := ValidateMap(&td)
		if err == nil {
			t.Fatalf("Didn't return error on missing mandatory title.")
		}
	})
}

package catalog

import (
	"testing"
)

func TestValidateThingDescription(t *testing.T) {
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
		err := validateThingDescription(td)
		if err == nil {
			t.Fatalf("Didn't return error on non-URI ID (%s): %s", td["id"], err)
		}
	})

	t.Run("non-float TTL", func(t *testing.T) {
		var td = map[string]any{
			"@context": "https://www.w3.org/2019/wot/td/v1",
			"id":       "urn:example:test/thing1",
			"title":    "example thing",
			"security": []string{"basic_sc"},
			"securityDefinitions": map[string]any{
				"basic_sc": map[string]string{
					"in":     "header",
					"scheme": "basic",
				},
			},
			"ttl": 1,
		}
		err := validateThingDescription(td)
		if err == nil {
			t.Fatalf("Didn't return error on integer TTL: %s", err)
		}

		td["ttl"] = "1"
		err = validateThingDescription(td)
		if err == nil {
			t.Fatalf("Didn't return error on string TTL: %s", err)
		}
	})
}

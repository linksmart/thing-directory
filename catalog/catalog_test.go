package catalog

import (
	"testing"
)

// here are only the tests related to non-standard TD vocabulary
func TestValidateThingDescription(t *testing.T) {
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
			t.Fatalf("Didn't return error on integer TTL.")
		}

		td["ttl"] = "1"
		err = validateThingDescription(td)
		if err == nil {
			t.Fatalf("Didn't return error on string TTL.")
		}
	})
}

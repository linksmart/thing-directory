package catalog

import (
	"testing"
)

// here are only the tests related to non-standard TD vocabulary
func TestValidateThingDescription(t *testing.T) {
	err := loadSchema()
	if err != nil {
		t.Fatalf("error loading WoT Thing Description schema: %s", err)
	}

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
			"registration": map[string]any{
				"ttl": "60",
			},
		}
		results, err := validateThingDescription(td)
		if err != nil {
			t.Fatalf("internal validation error: %s", err)
		}
		if len(results) == 0 {
			t.Fatalf("Didn't return error on string TTL.")
		}
	})
}

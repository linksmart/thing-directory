package main

import "testing"

func TestEscapeDNSSDServiceInstance(t *testing.T) {
	t.Run("no escaping", func(t *testing.T) {
		instance := "thing-directory"
		escaped := escapeDNSSDServiceInstance(instance)
		if escaped != instance {
			t.Fatalf("Unexpected escaping of %s to %s", instance, escaped)
		}
	})

	t.Run("escape dot", func(t *testing.T) {
		instance := "thing.directory"   // from thing.directory
		expected := "thing\\.directory" // to   thing\.directory
		escaped := escapeDNSSDServiceInstance(instance)
		if escaped != expected {
			t.Fatalf("Escaped value for %s is %s. Expected %s", instance, escaped, expected)
		}
	})

	t.Run("escape backslash", func(t *testing.T) {
		instance := "thing\\directory"   // from thing\directory
		expected := "thing\\\\directory" // to   thing\\directory
		escaped := escapeDNSSDServiceInstance(instance)
		if escaped != expected {
			t.Fatalf("Escaped value for %s is %s. Expected %s", instance, escaped, expected)
		}
	})
}

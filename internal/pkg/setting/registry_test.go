package setting

import "testing"

func TestResourceDomainDefinition(t *testing.T) {
	for _, item := range registry {
		if item.Key != ResourceDomainKey {
			continue
		}
		if item.Value != "" {
			t.Fatalf("default value = %q, want empty", item.Value)
		}
		if !item.IsPublic {
			t.Fatal("resource domain must be public")
		}
		return
	}
	t.Fatalf("%s is not registered", ResourceDomainKey)
}

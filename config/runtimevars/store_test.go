package runtimevars

import "testing"

func TestReplaceCopiesAndRemovesValues(t *testing.T) {
	input := map[string]string{"api_key": "first"}
	Replace(input)
	input["api_key"] = "mutated"

	value, err := Get("api_key")
	if err != nil || value != "first" {
		t.Fatalf("Get(api_key) = %q, %v; want first", value, err)
	}

	Replace(map[string]string{"region": "us-east"})
	if _, err := Get("api_key"); err == nil {
		t.Fatal("Replace retained a value absent from the new snapshot")
	}
	copy := LoadAll()
	copy["region"] = "changed"
	value, _ = Get("region")
	if value != "us-east" {
		t.Fatalf("LoadAll returned an alias; stored value is %q", value)
	}
}

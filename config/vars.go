package config

import (
	"strings"

	"squadron/config/runtimevars"
)

// ReplaceRuntimeVars installs the complete variable snapshot supplied by the
// Command Center. The worker never persists these values locally.
func ReplaceRuntimeVars(values map[string]string) { runtimevars.Replace(values) }

func LoadVars() map[string]string { return runtimevars.LoadAll() }

func GetVar(name string) (string, error) { return runtimevars.Get(name) }

func ResolveVariableValue(v *Variable) (string, error) {
	if value, err := runtimevars.Get(v.Name); err == nil {
		return value, nil
	}
	return v.Default, nil
}

func ResolveVarRef(ref string) (string, error) {
	if !strings.HasPrefix(ref, "var.") {
		return ref, nil
	}
	return runtimevars.Get(strings.TrimPrefix(ref, "var."))
}

package git

import "strings"

// ConfigGet reads a scoped config value without logging.
func ConfigGet(scope, key string) string {
	out, err := Output("config", scope, "--get", key)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// ConfigLocalGet reads a local config value without logging.
func ConfigLocalGet(key string) string {
	return ConfigGet("--local", key)
}

// ConfigGlobalGet reads a global config value without logging.
func ConfigGlobalGet(key string) string {
	return ConfigGet("--global", key)
}

// ConfigEffectiveLocal returns local value, or global if local is unset.
func ConfigEffectiveLocal(key string) string {
	if v := ConfigLocalGet(key); v != "" {
		return v
	}
	return ConfigGlobalGet(key)
}

// ConfigSet sets a scoped config value and logs the git invocation.
func ConfigSet(scope, key, value string) error {
	return Verbose("config", scope, key, value)
}

// ConfigLocalSet sets a local config value and logs the git invocation.
func ConfigLocalSet(key, value string) error {
	return ConfigSet("--local", key, value)
}

// ConfigUnset removes a scoped config key when present; logs the git invocation.
func ConfigUnset(scope, key string) error {
	if ConfigGet(scope, key) == "" {
		return nil
	}
	return Verbose("config", scope, "--unset", key)
}

// ConfigLocalUnset removes a local config key when present; logs the git invocation.
func ConfigLocalUnset(key string) error {
	return ConfigUnset("--local", key)
}

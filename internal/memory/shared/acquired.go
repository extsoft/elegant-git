package shared

// Acquired returns the global install marker from shared memory (empty if unset).
func Acquired(s *State) string {
	if s == nil {
		return ""
	}
	return s.AcquiredVersion
}

// SetAcquired records the global install marker in shared memory.
func SetAcquired(s *State, version string) {
	if s == nil {
		return
	}
	s.AcquiredVersion = version
}

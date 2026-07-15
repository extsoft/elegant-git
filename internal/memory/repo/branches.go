package repo

// BranchSource returns the recorded source ref for branch, or "" if unset.
func BranchSource(s *State, branch string) string {
	if s == nil || len(s.BranchSources) == 0 {
		return ""
	}
	return s.BranchSources[branch]
}

// SetBranchSource records the ref branch was created from.
func SetBranchSource(s *State, branch, source string) {
	if s.BranchSources == nil {
		s.BranchSources = map[string]string{}
	}
	s.BranchSources[branch] = source
}

// SetBranchSourceFromCWD records a branch source in the current repo memory.
func SetBranchSourceFromCWD(branch, source string) error {
	s, gitDir, err := LoadFromCWD()
	if err != nil {
		return err
	}
	SetBranchSource(s, branch, source)
	return Save(gitDir, s)
}

// ClearBranchSource removes a branch source entry.
func ClearBranchSource(s *State, branch string) {
	if s == nil || len(s.BranchSources) == 0 {
		return
	}
	delete(s.BranchSources, branch)
}

// ClearBranchSourceFromCWD removes a branch source entry from the current repo memory.
func ClearBranchSourceFromCWD(branch string) error {
	s, gitDir, err := LoadFromCWD()
	if err != nil {
		return err
	}
	ClearBranchSource(s, branch)
	return Save(gitDir, s)
}

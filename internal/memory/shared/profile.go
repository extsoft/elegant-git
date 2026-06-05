package shared

import (
	"fmt"
	"strings"

	"github.com/bees-hive/elegant-git/internal/uuidv7"
)

// ListProfiles returns all profiles keyed by id.
func ListProfiles(s *State) map[string]*Profile {
	if s == nil || s.Profiles == nil {
		return map[string]*Profile{}
	}
	return s.Profiles
}

// GetProfile returns a profile by id.
func GetProfile(s *State, id string) (*Profile, error) {
	p, ok := s.Profiles[id]
	if !ok || p == nil {
		return nil, ErrNotFound
	}
	return p, nil
}

// GetProfileByName returns profile id and profile for a display name.
func GetProfileByName(s *State, name string) (string, *Profile, error) {
	for id, p := range s.Profiles {
		if p != nil && p.Name == name {
			return id, p, nil
		}
	}
	return "", nil, ErrNotFound
}

// GetProfileByUserEmail returns profile id matching user email.
func GetProfileByUserEmail(s *State, email string) (string, *Profile, error) {
	for id, p := range s.Profiles {
		if p != nil && p.UserEmail == email {
			return id, p, nil
		}
	}
	return "", nil, ErrNotFound
}

// FindProfileByFields returns a profile matching all identity fields.
func FindProfileByFields(s *State, userName, userEmail, signingKey, editor, gpgProgram string) (string, *Profile, bool) {
	for id, p := range s.Profiles {
		if p != nil && p.UserName == userName && p.UserEmail == userEmail &&
			p.SigningKey == signingKey && p.Editor == editor && p.GPGProgram == gpgProgram {
			return id, p, true
		}
	}
	return "", nil, false
}

// CreateProfileInput holds fields for a new profile.
type CreateProfileInput struct {
	Name       string
	UserName   string
	UserEmail  string
	SigningKey string
	Editor     string
	GPGProgram string
}

// CreateProfile adds a profile and returns its id.
func CreateProfile(s *State, in CreateProfileInput) (string, error) {
	if in.Name == "" || in.UserName == "" || in.UserEmail == "" {
		return "", fmt.Errorf("name, user_name, and user_email are required")
	}
	for _, p := range s.Profiles {
		if p != nil && p.Name == in.Name {
			return "", fmt.Errorf("profile %q already exists", in.Name)
		}
	}
	id, err := uuidv7.New()
	if err != nil {
		return "", err
	}
	s.Profiles[id] = &Profile{
		Name:        in.Name,
		UserName:    in.UserName,
		UserEmail:   in.UserEmail,
		SigningKey:  in.SigningKey,
		Editor:      in.Editor,
		GPGProgram:  in.GPGProgram,
		LinkedRepos: []string{},
	}
	return id, nil
}

// UpdateProfileInput holds optional profile field updates.
type UpdateProfileInput struct {
	UserName   *string
	UserEmail  *string
	SigningKey *string
	Editor     *string
	GPGProgram *string
}

// UpdateProfile updates profile fields by id.
func UpdateProfile(s *State, id string, in UpdateProfileInput) error {
	p, err := GetProfile(s, id)
	if err != nil {
		return err
	}
	if in.UserName != nil {
		p.UserName = *in.UserName
	}
	if in.UserEmail != nil {
		p.UserEmail = *in.UserEmail
	}
	if in.SigningKey != nil {
		p.SigningKey = *in.SigningKey
	}
	if in.Editor != nil {
		p.Editor = *in.Editor
	}
	if in.GPGProgram != nil {
		p.GPGProgram = *in.GPGProgram
	}
	return nil
}

// DeleteProfile removes a profile when it has no linked repositories.
func DeleteProfile(s *State, id string) error {
	p, err := GetProfile(s, id)
	if err != nil {
		return err
	}
	if len(p.LinkedRepos) > 0 {
		names := linkedRepoNames(s, p.LinkedRepos)
		return fmt.Errorf("profile %q is linked to %d repository (-ies): %s.\nRun `git elegant repo configure <other>` in each (or `git elegant repo sync`) to relink before deleting",
			p.Name, len(p.LinkedRepos), strings.Join(names, ", "))
	}
	delete(s.Profiles, id)
	return nil
}

func linkedRepoNames(s *State, ids []string) []string {
	var names []string
	for _, id := range ids {
		if r, err := GetRepo(s, id); err == nil && r != nil {
			names = append(names, r.Name)
		} else {
			names = append(names, id)
		}
	}
	return names
}

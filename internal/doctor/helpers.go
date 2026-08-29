package doctor

import (
	"github.com/bees-hive/elegant-git/internal/git"
)

func suggestWorkspaceName() string {
	email := git.ConfigEffectiveLocal("user.email")
	for i := 0; i < len(email); i++ {
		if email[i] == '@' && i > 0 {
			return email[:i]
		}
	}
	return email
}

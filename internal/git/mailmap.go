package git

import (
	"context"
	"strings"
)

// CheckMailmap resolves "Name <email>" through the repository mailmap.
func CheckMailmap(ctx context.Context, repoPath, name, email string) (resolvedName, resolvedEmail string, err error) {
	ident := strings.TrimSpace(name) + " <" + strings.TrimSpace(email) + ">"
	out, err := RunGit(ctx, repoPath, "check-mailmap", ident)
	if err != nil {
		return name, email, err
	}
	return parseIdent(out)
}

func parseIdent(s string) (name, email string, err error) {
	s = strings.TrimSpace(s)
	start := strings.LastIndexByte(s, '<')
	end := strings.LastIndexByte(s, '>')
	if start < 0 || end <= start {
		return s, "", nil
	}
	name = strings.TrimSpace(s[:start])
	email = strings.TrimSpace(s[start+1 : end])
	return name, email, nil
}

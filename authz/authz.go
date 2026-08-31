// Package authz holds the types shared by the policy tables that
// protoc-gen-authz-go generates and by the interceptor that enforces them. It
// depends on the standard library only, so importing a generated table never
// pulls an RPC framework into a caller.
package authz

import (
	"errors"
	"fmt"
)

// Level is the access level a procedure declares. Values match authz.v1.AuthLevel.
type Level int

const (
	// LevelUnspecified is fail-closed and is generated as LevelAuthenticated.
	LevelUnspecified Level = 0
	// LevelPublic permits unauthenticated callers.
	LevelPublic Level = 1
	// LevelAuthenticated requires an authenticated caller.
	LevelAuthenticated Level = 2
	// LevelInternal is for service-to-service procedures and must not be
	// exposed externally by a gateway.
	LevelInternal Level = 3
)

// String returns the lower-case name of the level, or "Level(n)" for a value
// this package does not know.
func (l Level) String() string {
	switch l {
	case LevelUnspecified:
		return "unspecified"
	case LevelPublic:
		return "public"
	case LevelAuthenticated:
		return "authenticated"
	case LevelInternal:
		return "internal"
	default:
		return fmt.Sprintf("Level(%d)", int(l))
	}
}

// Policy is what a procedure requires of its caller.
type Policy struct {
	Level Level
	// RequiredScopes must all be granted to the caller.
	RequiredScopes []string
	// TokenUses are the token_use values the procedure accepts. Empty leaves the
	// choice to the interceptor's defaults for the level.
	TokenUses []string
}

// Policies maps a procedure ("/pkg.Service/Method") to its effective policy.
type Policies map[string]Policy

// Lookup returns the policy of procedure.
func (p Policies) Lookup(procedure string) (Policy, bool) {
	policy, ok := p[procedure]
	return policy, ok
}

// ErrDuplicateProcedure reports that more than one table names the same
// procedure. Merge wraps it so that errors.Is identifies the condition.
var ErrDuplicateProcedure = errors.New("authz: duplicate procedure")

// Merge combines the tables of several services served by one process. It fails
// on a procedure named by more than one table.
func Merge(tables ...Policies) (Policies, error) {
	merged := make(Policies)
	for _, table := range tables {
		for procedure, policy := range table {
			if _, ok := merged[procedure]; ok {
				return nil, fmt.Errorf("%w: %s", ErrDuplicateProcedure, procedure)
			}
			merged[procedure] = policy
		}
	}
	return merged, nil
}

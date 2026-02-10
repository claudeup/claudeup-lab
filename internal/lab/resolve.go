package lab

import (
	"fmt"
	"strings"
)

// Resolver finds labs by fuzzy matching against stored metadata.
type Resolver struct {
	store *StateStore
}

func NewResolver(store *StateStore) *Resolver {
	return &Resolver{store: store}
}

// Resolve finds a lab by exact UUID, display name, partial UUID prefix,
// project name, or profile name.
func (r *Resolver) Resolve(query string) (*Metadata, error) {
	// Exact UUID match
	if meta, err := r.store.Load(query); err == nil {
		return meta, nil
	}

	labs, err := r.store.List()
	if err != nil {
		return nil, fmt.Errorf("list labs: %w", err)
	}

	var matches []*Metadata
	for _, m := range labs {
		// Display name match (exact, return immediately)
		if m.DisplayName == query {
			return m, nil
		}

		// Partial UUID prefix
		if strings.HasPrefix(m.ID, query) {
			matches = append(matches, m)
			continue
		}

		// Project name match
		if m.ProjectName == query {
			matches = append(matches, m)
			continue
		}

		// Profile match
		if m.Profile == query {
			matches = append(matches, m)
			continue
		}
	}

	if len(matches) == 1 {
		return matches[0], nil
	}

	if len(matches) > 1 {
		return nil, &AmbiguousError{Query: query, Matches: matches}
	}

	return nil, &NotFoundError{Query: query, Available: labs}
}

// ResolveByCWD finds a lab whose worktree is a parent of the given path.
func (r *Resolver) ResolveByCWD(cwd string) (*Metadata, error) {
	labs, err := r.store.List()
	if err != nil {
		return nil, fmt.Errorf("list labs: %w", err)
	}

	for _, m := range labs {
		if strings.HasPrefix(cwd, m.Worktree) {
			return m, nil
		}
	}

	return nil, &NotFoundError{Query: cwd, Available: labs}
}

// AmbiguousError indicates multiple labs matched a query.
type AmbiguousError struct {
	Query   string
	Matches []*Metadata
}

func (e *AmbiguousError) Error() string {
	var names []string
	for _, m := range e.Matches {
		names = append(names, fmt.Sprintf("%s (%s)", m.DisplayName, shortID(m.ID)))
	}
	return fmt.Sprintf("ambiguous lab query %q, matches: %s", e.Query, strings.Join(names, ", "))
}

// NotFoundError indicates no labs matched a query.
type NotFoundError struct {
	Query     string
	Available []*Metadata
}

func (e *NotFoundError) Error() string {
	if len(e.Available) == 0 {
		return fmt.Sprintf("no lab matched %q (no labs found)", e.Query)
	}
	var names []string
	for _, m := range e.Available {
		names = append(names, fmt.Sprintf("%s (%s)", m.DisplayName, shortID(m.ID)))
	}
	return fmt.Sprintf("no lab matched %q, available: %s", e.Query, strings.Join(names, ", "))
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

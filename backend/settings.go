package backend

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"aion/backend/model"
	"aion/backend/store"
)

// UpdateCheckPolicy controls background silent update checks.
type UpdateCheckPolicy struct {
	Enabled  bool          `json:"enabled"`
	Interval time.Duration `json:"interval"`
}

const (
	defaultUpdateCheckInterval = 24 * time.Hour
	minUpdateCheckInterval     = time.Minute
	maxUpdateCheckInterval     = 365 * 24 * time.Hour
)

// DefaultUpdateCheckPolicy returns the default silent update check preference.
func DefaultUpdateCheckPolicy() UpdateCheckPolicy {
	return UpdateCheckPolicy{
		Enabled:  true,
		Interval: defaultUpdateCheckInterval,
	}
}

// Settings stores versionable module preferences. Keys are namespaced, e.g.
// app.theme or projects.card_density. Backup policy is owned by backup.Service.
type Settings struct{ s *store.Store }

func (s *Settings) Get(ctx context.Context, key string, out any) error {
	if !validKey(key) {
		return model.ErrInvalid
	}
	var value string
	err := s.s.Read(ctx, func(tx *sql.Tx) error {
		if err := tx.QueryRowContext(ctx, "SELECT value FROM settings WHERE key=?", key).Scan(&value); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return model.ErrNotFound
			}
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(value), out)
}
func (s *Settings) Set(ctx context.Context, key string, value any) error {
	if !validKey(key) {
		return model.ErrInvalid
	}
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(b) > 1024*1024 {
		return fmt.Errorf("%w: setting too large", model.ErrInvalid)
	}
	return s.s.Write(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", key, string(b))
		return err
	})
}
func (s *Settings) Theme(ctx context.Context) (string, error) {
	var value string
	err := s.Get(ctx, "app.theme", &value)
	if errors.Is(err, model.ErrNotFound) {
		return "system", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
}
func (s *Settings) SetTheme(ctx context.Context, theme string) error {
	if theme != "light" && theme != "dark" && theme != "system" {
		return model.ErrInvalid
	}
	return s.Set(ctx, "app.theme", theme)
}
func (s *Settings) UpdateCheckPolicy(ctx context.Context) (UpdateCheckPolicy, error) {
	var pol UpdateCheckPolicy
	err := s.Get(ctx, "app.update_check", &pol)
	if errors.Is(err, model.ErrNotFound) {
		return DefaultUpdateCheckPolicy(), nil
	}
	if err != nil {
		return UpdateCheckPolicy{}, err
	}
	if pol.Interval < minUpdateCheckInterval || pol.Interval > maxUpdateCheckInterval {
		pol.Interval = defaultUpdateCheckInterval
	}
	return pol, nil
}
func (s *Settings) SetUpdateCheckPolicy(ctx context.Context, pol UpdateCheckPolicy) error {
	if pol.Interval < minUpdateCheckInterval || pol.Interval > maxUpdateCheckInterval {
		return fmt.Errorf("%w: update check interval must be between 1m and 365d", model.ErrInvalid)
	}
	return s.Set(ctx, "app.update_check", pol)
}

// HiddenModules returns the sorted, deduplicated IDs of modules hidden in the workbench UI.
// Hiding a module never deletes its data; newly added built-in modules are visible by default.
func (s *Settings) HiddenModules(ctx context.Context) ([]string, error) {
	var raw []string
	err := s.Get(ctx, "app.hidden_modules", &raw)
	if errors.Is(err, model.ErrNotFound) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	return normalizeModuleIDs(raw)
}

// SetHiddenModules persists the list of hidden module IDs.
func (s *Settings) SetHiddenModules(ctx context.Context, ids []string) error {
	norm, err := normalizeModuleIDs(ids)
	if err != nil {
		return err
	}
	return s.Set(ctx, "app.hidden_modules", norm)
}

func normalizeModuleIDs(ids []string) ([]string, error) {
	out := make([]string, 0, len(ids))
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if !validModuleID(id) {
			return nil, fmt.Errorf("%w: invalid module id %q", model.ErrInvalid, id)
		}
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	slices.Sort(out)
	return out, nil
}
func validKey(key string) bool {
	if len(key) == 0 || len(key) > 128 || !strings.Contains(key, ".") {
		return false
	}
	for _, r := range key {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

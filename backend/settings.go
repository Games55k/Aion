package backend

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"aion/backend/model"
	"aion/backend/store"
)

// Settings stores versionable module preferences. Keys are namespaced, e.g.
// app.theme or projects.card_density. Backup policy is owned by backup.Service.
type Settings struct{ s *store.Store }

func (s *Settings) Get(ctx context.Context, key string, out any) error {
	if !validKey(key) {
		return model.ErrInvalid
	}
	return s.s.Read(ctx, func(tx *sql.Tx) error {
		var value string
		if err := tx.QueryRowContext(ctx, "SELECT value FROM settings WHERE key=?", key).Scan(&value); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return model.ErrNotFound
			}
			return err
		}
		return json.Unmarshal([]byte(value), out)
	})
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

package backend

import (
	"fmt"
	"sync"

	"aion/backend/model"
)

// ModuleDescriptor describes a compiled-in tool, not an executable plugin.
type ModuleDescriptor struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version int    `json:"version"`
}

// Registry accepts independent built-in tool descriptors during startup.
// Modules own their services and database migrations; registration never runs
// untrusted code or forces another tool to adopt the project/task model.
type Registry struct {
	mu      sync.RWMutex
	modules []ModuleDescriptor
}

func (r *Registry) Register(m ModuleDescriptor) error {
	if !validModuleID(m.ID) || m.Name == "" || m.Version < 1 {
		return model.ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, old := range r.modules {
		if old.ID == m.ID {
			return fmt.Errorf("%w: module %s", model.ErrConflict, m.ID)
		}
	}
	r.modules = append(r.modules, m)
	return nil
}
func (r *Registry) List() []ModuleDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]ModuleDescriptor{}, r.modules...)
}
func validModuleID(s string) bool {
	if len(s) == 0 || len(s) > 64 {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_') {
			return false
		}
	}
	return true
}

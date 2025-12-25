package actions

import (
	"encoding/json"
	"fmt"
	"sort"
	"whatsabladerunner/pkg/behaviors"
	"whatsabladerunner/pkg/tasks"
)

// ActionSchema describes the action for the LLM
type ActionSchema struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"` // JSON Schema object
}

// ActionContext holds execution context
type ActionContext struct {
	Context         []string
	Task            *tasks.Task // nil if not in task mode
	BehaviorManager *behaviors.BehaviorManager
	SendToContact   func(string)
	ToolOutputs     *[]string // New: For returning tool results to the loop
}

// Action represents an executable action
type Action interface {
	GetSchema() ActionSchema
	Execute(ctx ActionContext, payload json.RawMessage) error
}

// Registry manages available actions
type Registry struct {
	actions map[string]Action
}

func NewRegistry() *Registry {
	return &Registry{
		actions: make(map[string]Action),
	}
}

func (r *Registry) Register(a Action) {
	schema := a.GetSchema()
	r.actions[schema.Name] = a
}

func (r *Registry) Get(name string) (Action, bool) {
	a, ok := r.actions[name]
	return a, ok
}

// GetSchemas returns a list of schemas for all registered actions, sorted by name
func (r *Registry) GetSchemas() []ActionSchema {
	return r.GetSchemasFiltered(nil)
}

// GetSchemasFiltered returns a list of schemas for actions NOT in the exclude list, sorted by name
func (r *Registry) GetSchemasFiltered(exclude []string) []ActionSchema {
	schemas := make([]ActionSchema, 0, len(r.actions))
	for name, a := range r.actions {
		excluded := false
		for _, e := range exclude {
			if e == name {
				excluded = true
				break
			}
		}
		if !excluded {
			schemas = append(schemas, a.GetSchema())
		}
	}
	// Sort for deterministic output
	sort.Slice(schemas, func(i, j int) bool {
		return schemas[i].Name < schemas[j].Name
	})
	return schemas
}

// Validate checks if an action exists
func (r *Registry) Validate(name string) error {
	if _, ok := r.actions[name]; !ok {
		return fmt.Errorf("action '%s' is not registered", name)
	}
	return nil
}

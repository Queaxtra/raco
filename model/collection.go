package model

import "time"

type Collection struct {
	ID        string            `json:"id" yaml:"id"`
	Name      string            `json:"name" yaml:"name"`
	Tags      []string          `json:"tags,omitempty" yaml:"tags,omitempty"`
	Hooks     CollectionHooks   `json:"hooks,omitempty" yaml:"hooks,omitempty"`
	Contracts []ContractProfile `json:"contracts,omitempty" yaml:"contracts,omitempty"`
	Requests  []*Request        `json:"requests" yaml:"requests"`
	Revision  int               `json:"revision,omitempty" yaml:"revision,omitempty"`
	UpdatedAt time.Time         `json:"updated_at,omitempty" yaml:"updated_at,omitempty"`
}

package model

type Collection struct {
	ID       string     `json:"id" yaml:"id"`
	Name     string     `json:"name" yaml:"name"`
	Requests []*Request `json:"requests" yaml:"requests"`
}

package model

import "time"

type CollectionHooks struct {
	Setup    []string `json:"setup,omitempty" yaml:"setup,omitempty"`
	Teardown []string `json:"teardown,omitempty" yaml:"teardown,omitempty"`
}

type ContractProfile struct {
	Name       string      `json:"name" yaml:"name"`
	Assertions []Assertion `json:"assertions,omitempty" yaml:"assertions,omitempty"`
	Headers    []string    `json:"headers,omitempty" yaml:"headers,omitempty"`
	Schema     string      `json:"schema,omitempty" yaml:"schema,omitempty"`
}

type TransportConfig struct {
	ProxyURL        string `json:"proxy_url,omitempty" yaml:"proxy_url,omitempty"`
	CAFile          string `json:"ca_file,omitempty" yaml:"ca_file,omitempty"`
	CertFile        string `json:"cert_file,omitempty" yaml:"cert_file,omitempty"`
	KeyFile         string `json:"key_file,omitempty" yaml:"key_file,omitempty"`
	AllowInsecure   bool   `json:"allow_insecure,omitempty" yaml:"allow_insecure,omitempty"`
	AllowPrivateIP  bool   `json:"allow_private_ip,omitempty" yaml:"allow_private_ip,omitempty"`
	RateLimitPerMin int    `json:"rate_limit_per_min,omitempty" yaml:"rate_limit_per_min,omitempty"`
}

type RetryPolicy struct {
	MaxAttempts        int  `json:"max_attempts,omitempty" yaml:"max_attempts,omitempty"`
	BaseDelayMS        int  `json:"base_delay_ms,omitempty" yaml:"base_delay_ms,omitempty"`
	MaxDelayMS         int  `json:"max_delay_ms,omitempty" yaml:"max_delay_ms,omitempty"`
	RetryNonIdempotent bool `json:"retry_non_idempotent,omitempty" yaml:"retry_non_idempotent,omitempty"`
}

type SnapshotConfig struct {
	Enabled  bool   `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Name     string `json:"name,omitempty" yaml:"name,omitempty"`
	FilePath string `json:"file_path,omitempty" yaml:"file_path,omitempty"`
}

type ProtocolConfig struct {
	ScriptFile string            `json:"script_file,omitempty" yaml:"script_file,omitempty"`
	Service    string            `json:"service,omitempty" yaml:"service,omitempty"`
	Method     string            `json:"method,omitempty" yaml:"method,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type ProtocolScript struct {
	Name     string               `json:"name" yaml:"name"`
	Protocol string               `json:"protocol" yaml:"protocol"`
	Steps    []ProtocolScriptStep `json:"steps" yaml:"steps"`
}

type ProtocolScriptStep struct {
	Type        string              `json:"type" yaml:"type"`
	Send        string              `json:"send,omitempty" yaml:"send,omitempty"`
	Expect      string              `json:"expect,omitempty" yaml:"expect,omitempty"`
	TimeoutMS   int                 `json:"timeout_ms,omitempty" yaml:"timeout_ms,omitempty"`
	Assertions  []ProtocolAssertion `json:"assertions,omitempty" yaml:"assertions,omitempty"`
	Extractors  []ProtocolExtractor `json:"extractors,omitempty" yaml:"extractors,omitempty"`
	Description string              `json:"description,omitempty" yaml:"description,omitempty"`
}

type ProtocolAssertion struct {
	Type     string `json:"type" yaml:"type"`
	Operator string `json:"operator" yaml:"operator"`
	Value    string `json:"value" yaml:"value"`
}

type ProtocolExtractor struct {
	Type    string `json:"type" yaml:"type"`
	Source  string `json:"source" yaml:"source"`
	Target  string `json:"target" yaml:"target"`
	Pattern string `json:"pattern,omitempty" yaml:"pattern,omitempty"`
}

type CollectionRevision struct {
	CollectionID string     `json:"collection_id"`
	Revision     int        `json:"revision"`
	UpdatedAt    time.Time  `json:"updated_at"`
	Collection   Collection `json:"collection"`
}

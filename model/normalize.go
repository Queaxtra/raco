package model

import (
	"strings"
	"time"
)

func NormalizeCollection(col *Collection) {
	if col == nil {
		return
	}

	if col.Requests == nil {
		col.Requests = make([]*Request, 0)
	}
	col.Tags = normalizeTags(col.Tags)
	col.Hooks.Setup = normalizeRefs(col.Hooks.Setup)
	col.Hooks.Teardown = normalizeRefs(col.Hooks.Teardown)
	if col.Contracts == nil {
		col.Contracts = make([]ContractProfile, 0)
	}
	if col.Revision < 0 {
		col.Revision = 0
	}
	for _, req := range col.Requests {
		NormalizeRequest(req)
		if req == nil {
			continue
		}
		if req.CollectionID == "" {
			req.CollectionID = col.ID
		}
	}
}

func NormalizeRequest(req *Request) {
	if req == nil {
		return
	}

	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	if req.Query == nil {
		req.Query = make(map[string]string)
	}
	if req.Files == nil {
		req.Files = make([]FileUpload, 0)
	}
	if req.Assertions == nil {
		req.Assertions = make([]Assertion, 0)
	}
	if req.Extractors == nil {
		req.Extractors = make([]Extractor, 0)
	}
	req.Tags = normalizeTags(req.Tags)
	if req.Protocol.Metadata == nil {
		req.Protocol.Metadata = make(map[string]string)
	}
}

func NormalizeEnvironment(env *Environment) {
	if env == nil {
		return
	}

	if env.Variables == nil {
		env.Variables = make(map[string]EnvironmentVariable)
	}
	env.Parent = strings.TrimSpace(env.Parent)
	for key, variable := range env.Variables {
		if variable.Kind != "" {
			continue
		}
		env.Variables[key] = EnvironmentVariable{
			Kind:  EnvironmentVariablePlain,
			Value: variable.Value,
			Ref:   variable.Ref,
		}
	}
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return make([]string, 0)
	}

	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(strings.ToLower(tag))
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}
	return out
}

func normalizeRefs(refs []string) []string {
	if len(refs) == 0 {
		return make([]string, 0)
	}

	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		out = append(out, ref)
	}
	return out
}

func (c *Collection) Touch() {
	if c == nil {
		return
	}

	c.Revision++
	c.UpdatedAt = time.Now().UTC()
}

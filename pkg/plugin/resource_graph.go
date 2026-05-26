package plugin

import "context"

type ResourceGraphRequest struct {
	Connection map[string]string `json:"connection,omitempty"`
	ResourceID string            `json:"resource_id,omitempty"`
	Depth      int               `json:"depth,omitempty"`
}

// ResourceGraphProvider is implemented by plugins that expose a generic
// browse tree via the resource-graph CLI command.
type ResourceGraphProvider interface {
	ResourceGraph(ctx context.Context, req *ResourceGraphRequest) (*ResourceGraphResponse, error)
}

type ResourceGraphResponse struct {
	Nodes []*ResourceNode `json:"nodes,omitempty"`
}

type ResourceNode struct {
	ID       string            `json:"id,omitempty"`
	Name     string            `json:"name,omitempty"`
	Kind     string            `json:"kind,omitempty"`
	Path     string            `json:"path,omitempty"`
	Actions  []*ResourceAction `json:"actions,omitempty"`
	Children []*ResourceNode   `json:"children,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type ResourceAction struct {
	ID       string            `json:"id,omitempty"`
	Kind     string            `json:"kind,omitempty"`
	Title    string            `json:"title,omitempty"`
	Query    string            `json:"query,omitempty"`
	NewTab   bool              `json:"new_tab,omitempty"`
	Fields   []*ResourceField  `json:"fields,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type ResourceField struct {
	Name        string `json:"name,omitempty"`
	Label       string `json:"label,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

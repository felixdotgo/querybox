package plugin

import (
	"context"

	pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"
)

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

// AdaptConnectionTree converts the legacy connection-tree payload into the
// resource-graph shape used by the runtime and frontend explorer.
func AdaptConnectionTree(tree *ConnectionTreeResponse) *ResourceGraphResponse {
	resp := &ResourceGraphResponse{}
	if tree == nil {
		return resp
	}
	resp.Nodes = make([]*ResourceNode, 0, len(tree.Nodes))
	for _, node := range tree.Nodes {
		if node == nil {
			continue
		}
		resp.Nodes = append(resp.Nodes, adaptConnectionTreeNode(node))
	}
	return resp
}

// AdaptResourceGraph converts a resource graph into the older tree shape.
func AdaptResourceGraph(graph *ResourceGraphResponse) *ConnectionTreeResponse {
	resp := &ConnectionTreeResponse{}
	if graph == nil {
		return resp
	}
	resp.Nodes = make([]*ConnectionTreeNode, 0, len(graph.Nodes))
	for _, node := range graph.Nodes {
		if node == nil {
			continue
		}
		resp.Nodes = append(resp.Nodes, adaptResourceNode(node))
	}
	return resp
}

func adaptConnectionTreeNode(node *ConnectionTreeNode) *ResourceNode {
	if node == nil {
		return nil
	}
	out := &ResourceNode{
		ID:       node.Key,
		Name:     node.Label,
		Kind:     nodeKind(node.NodeType),
		Path:     node.Key,
		Metadata: map[string]string{},
	}
	if out.Kind != "" {
		out.Metadata["legacy_node_type"] = out.Kind
	}
	for _, action := range node.Actions {
		if action == nil {
			continue
		}
		out.Actions = append(out.Actions, adaptConnectionTreeAction(action))
	}
	for _, child := range node.Children {
		if child == nil {
			continue
		}
		out.Children = append(out.Children, adaptConnectionTreeNode(child))
	}
	return out
}

func adaptResourceNode(node *ResourceNode) *ConnectionTreeNode {
	if node == nil {
		return nil
	}
	out := &ConnectionTreeNode{
		Key:      firstNonEmpty(node.Path, node.ID, node.Name),
		Label:    firstNonEmpty(node.Name, node.ID, node.Path),
		NodeType: legacyNodeType(node.Kind),
	}
	for _, action := range node.Actions {
		if action == nil {
			continue
		}
		out.Actions = append(out.Actions, adaptResourceAction(action))
	}
	for _, child := range node.Children {
		if child == nil {
			continue
		}
		out.Children = append(out.Children, adaptResourceNode(child))
	}
	return out
}

func adaptConnectionTreeAction(action *ConnectionTreeAction) *ResourceAction {
	if action == nil {
		return nil
	}
	return &ResourceAction{
		ID:       action.Type,
		Kind:     action.Type,
		Title:    action.Title,
		Query:    action.Query,
		NewTab:   action.NewTab,
		Metadata: map[string]string{},
	}
}

func adaptResourceAction(action *ResourceAction) *ConnectionTreeAction {
	if action == nil {
		return nil
	}
	return &ConnectionTreeAction{
		Type:   firstNonEmpty(action.Kind, action.ID),
		Title:  action.Title,
		Query:  action.Query,
		NewTab: action.NewTab,
	}
}

func nodeKind(nodeType pluginpb.PluginV1_NodeType) string {
	switch nodeType {
	case ConnectionTreeNodeTypeDatabase:
		return "database"
	case ConnectionTreeNodeTypeTable:
		return "table"
	case ConnectionTreeNodeTypeColumn:
		return "column"
	case ConnectionTreeNodeTypeSchema:
		return "schema"
	case ConnectionTreeNodeTypeView:
		return "view"
	case ConnectionTreeNodeTypeAction:
		return "action"
	case ConnectionTreeNodeTypeCollection:
		return "collection"
	case ConnectionTreeNodeTypeKey:
		return "key"
	case ConnectionTreeNodeTypeGroup:
		return "group"
	default:
		return "resource"
	}
}

func legacyNodeType(kind string) pluginpb.PluginV1_NodeType {
	switch kind {
	case "database":
		return ConnectionTreeNodeTypeDatabase
	case "table":
		return ConnectionTreeNodeTypeTable
	case "column":
		return ConnectionTreeNodeTypeColumn
	case "schema":
		return ConnectionTreeNodeTypeSchema
	case "view":
		return ConnectionTreeNodeTypeView
	case "action":
		return ConnectionTreeNodeTypeAction
	case "collection":
		return ConnectionTreeNodeTypeCollection
	case "key":
		return ConnectionTreeNodeTypeKey
	case "group":
		return ConnectionTreeNodeTypeGroup
	default:
		return 0
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

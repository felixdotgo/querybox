package pluginmgr

import (
	"encoding/json"
	"fmt"

	"github.com/felixdotgo/querybox/pkg/plugin"
	pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"
	"github.com/felixdotgo/querybox/services"
	"google.golang.org/protobuf/encoding/protojson"
)

func (m *Manager) GetResourceGraph(name string, connection map[string]string) (*plugin.ResourceGraphResponse, error) {
	req := plugin.ResourceGraphRequest{Connection: connection}
	payload, err := json.Marshal(&req)
	if err != nil {
		return nil, fmt.Errorf("GetResourceGraph: marshal request: %w", err)
	}

	if info, ok := m.lookupPlugin(name); ok && supportsCapability(info.Capabilities, "resource.graph") {
		out, execErr := m.runPluginCommand("GetResourceGraph", name, "resource-graph", defaultPluginTimeout, payload)
		if execErr == nil {
			resp := &plugin.ResourceGraphResponse{}
			if len(out) == 0 {
				return resp, nil
			}
			if err := json.Unmarshal(out, resp); err != nil {
				return nil, fmt.Errorf("GetResourceGraph: invalid graph json: %w", err)
			}
			return resp, nil
		}
		m.emitLog(services.LogLevelInfo, fmt.Sprintf("GetResourceGraph: falling back to legacy adapter for %s: %v", name, execErr))
	}

	tree, err := m.GetConnectionTree(name, connection)
	if err != nil {
		return nil, err
	}
	return adaptConnectionTree(tree), nil
}

func adaptConnectionTree(tree *plugin.ConnectionTreeResponse) *plugin.ResourceGraphResponse {
	resp := &plugin.ResourceGraphResponse{}
	if tree == nil {
		return resp
	}
	resp.Nodes = make([]*plugin.ResourceNode, 0, len(tree.Nodes))
	for _, node := range tree.Nodes {
		if node == nil {
			continue
		}
		resp.Nodes = append(resp.Nodes, adaptConnectionTreeNode(node))
	}
	return resp
}

func adaptResourceGraph(graph *plugin.ResourceGraphResponse) *plugin.ConnectionTreeResponse {
	resp := &plugin.ConnectionTreeResponse{}
	if graph == nil {
		return resp
	}
	resp.Nodes = make([]*plugin.ConnectionTreeNode, 0, len(graph.Nodes))
	for _, node := range graph.Nodes {
		if node == nil {
			continue
		}
		resp.Nodes = append(resp.Nodes, adaptResourceNode(node))
	}
	return resp
}

func adaptConnectionTreeNode(node *plugin.ConnectionTreeNode) *plugin.ResourceNode {
	if node == nil {
		return nil
	}
	out := &plugin.ResourceNode{
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

func adaptResourceNode(node *plugin.ResourceNode) *plugin.ConnectionTreeNode {
	if node == nil {
		return nil
	}
	out := &plugin.ConnectionTreeNode{
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

func adaptConnectionTreeAction(action *plugin.ConnectionTreeAction) *plugin.ResourceAction {
	if action == nil {
		return nil
	}
	return &plugin.ResourceAction{
		ID:       action.Type,
		Kind:     action.Type,
		Title:    action.Title,
		Query:    action.Query,
		NewTab:   action.NewTab,
		Metadata: map[string]string{},
	}
}

func adaptResourceAction(action *plugin.ResourceAction) *plugin.ConnectionTreeAction {
	if action == nil {
		return nil
	}
	return &plugin.ConnectionTreeAction{
		Type:   firstNonEmpty(action.Kind, action.ID),
		Title:  action.Title,
		Query:  action.Query,
		NewTab: action.NewTab,
	}
}

func nodeKind(nodeType any) string {
	switch value := fmt.Sprintf("%v", nodeType); value {
	case "NODE_TYPE_DATABASE", "1":
		return "database"
	case "NODE_TYPE_TABLE", "2":
		return "table"
	case "NODE_TYPE_COLUMN", "3":
		return "column"
	case "NODE_TYPE_SCHEMA", "4":
		return "schema"
	case "NODE_TYPE_VIEW", "5":
		return "view"
	case "NODE_TYPE_ACTION", "6":
		return "action"
	case "NODE_TYPE_COLLECTION", "7":
		return "collection"
	case "NODE_TYPE_KEY", "8":
		return "key"
	case "NODE_TYPE_GROUP", "9":
		return "group"
	default:
		return "resource"
	}
}

func supportsCapability(capabilities []string, capability string) bool {
	for _, current := range capabilities {
		if current == capability {
			return true
		}
	}
	return false
}

func decodeConnectionTreeJSON(raw []byte) (*plugin.ConnectionTreeResponse, error) {
	resp := &plugin.ConnectionTreeResponse{}
	if len(raw) == 0 {
		return resp, nil
	}
	if err := protojson.Unmarshal(raw, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func decodeResourceGraphJSON(raw []byte) (*plugin.ResourceGraphResponse, error) {
	resp := &plugin.ResourceGraphResponse{}
	if len(raw) == 0 {
		return resp, nil
	}
	if err := json.Unmarshal(raw, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func legacyNodeType(kind string) pluginpb.PluginV1_NodeType {
	switch kind {
	case "database":
		return pluginpb.PluginV1_NODE_TYPE_DATABASE
	case "table":
		return pluginpb.PluginV1_NODE_TYPE_TABLE
	case "column":
		return pluginpb.PluginV1_NODE_TYPE_COLUMN
	case "schema":
		return pluginpb.PluginV1_NODE_TYPE_SCHEMA
	case "view":
		return pluginpb.PluginV1_NODE_TYPE_VIEW
	case "action":
		return pluginpb.PluginV1_NODE_TYPE_ACTION
	case "collection":
		return pluginpb.PluginV1_NODE_TYPE_COLLECTION
	case "key":
		return pluginpb.PluginV1_NODE_TYPE_KEY
	case "group":
		return pluginpb.PluginV1_NODE_TYPE_GROUP
	default:
		return pluginpb.PluginV1_NODE_TYPE_UNKNOWN
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

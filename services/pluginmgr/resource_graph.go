package pluginmgr

import (
	"encoding/json"
	"fmt"

	"github.com/felixdotgo/querybox/pkg/plugin"
	"github.com/felixdotgo/querybox/services"
)

func (m *Manager) GetResourceGraph(name string, connection map[string]string) (*plugin.ResourceGraphResponse, error) {
	m.emitLog(services.LogLevelInfo, fmt.Sprintf("GetResourceGraph: fetching graph (driver: %s)", name))
	req := plugin.ResourceGraphRequest{Connection: connection}
	payload, err := json.Marshal(&req)
	if err != nil {
		return nil, fmt.Errorf("GetResourceGraph: marshal request: %w", err)
	}

	if info, ok := m.lookupPlugin(name); ok && !supportsCapability(info.Capabilities, "resource.graph") {
		return nil, fmt.Errorf("GetResourceGraph: plugin %s does not declare resource.graph capability", name)
	}

	out, err := m.runPluginCommand("GetResourceGraph", name, "resource-graph", defaultPluginTimeout, payload)
	if err != nil {
		return nil, err
	}
	resp, err := decodeResourceGraphJSON(out)
	if err != nil {
		return nil, fmt.Errorf("GetResourceGraph: invalid graph json: %w", err)
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

func supportsCapability(capabilities []string, capability string) bool {
	for _, current := range capabilities {
		if current == capability {
			return true
		}
	}
	return false
}

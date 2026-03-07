package plugin

import pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"

// Type aliases for the GetCompletionFields protobuf messages so plugin authors
// can import a single stable package (`pkg/plugin`) without referencing the
// generated rpc package directly.
type GetCompletionFieldsRequest = pluginpb.PluginV1_GetCompletionFieldsRequest
type GetCompletionFieldsResponse = pluginpb.PluginV1_GetCompletionFieldsResponse
type FieldInfo = pluginpb.PluginV1_FieldInfo

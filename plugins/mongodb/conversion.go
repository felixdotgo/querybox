package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/felixdotgo/querybox/pkg/plugin"
	pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/protobuf/types/known/structpb"
)

// bsonDocToStruct converts a bson.D document to a *structpb.Struct.
// It round-trips through relaxed extended JSON to handle ObjectID and other
// BSON-specific types safely.
func bsonDocToStruct(doc bson.D) (*structpb.Struct, error) {
    raw, err := bson.MarshalExtJSON(doc, false, false)
    if err != nil {
        return nil, fmt.Errorf("marshal ext-json: %w", err)
    }
    var m map[string]interface{}
    if err := json.Unmarshal(raw, &m); err != nil {
        return nil, fmt.Errorf("unmarshal to map: %w", err)
    }
    return structpb.NewStruct(m)
}

// kvResponse wraps a string map into a KeyValueResult ExecResponse.
func kvResponse(data map[string]string) *plugin.ExecResponse {
    return &plugin.ExecResponse{
        Result: &plugin.ExecResult{
            Payload: &pluginpb.PluginV1_ExecResult_Kv{
                Kv: &plugin.KeyValueResult{Data: data},
            },
        },
    }
}

// cursorToDocumentResponse drains a cursor and returns a DocumentResult response.
func cursorToDocumentResponse(ctx context.Context, cursor *mongo.Cursor) (*plugin.ExecResponse, error) {
    var docs []*structpb.Struct
    for cursor.Next(ctx) {
        var doc bson.D
        if err := cursor.Decode(&doc); err != nil {
            continue
        }
        s, err := bsonDocToStruct(doc)
        if err != nil {
            continue
        }
        docs = append(docs, s)
    }
    if err := cursor.Err(); err != nil {
        return &plugin.ExecResponse{Error: fmt.Sprintf("cursor error: %v", err)}, nil
    }
    if docs == nil {
        docs = []*structpb.Struct{}
    }
    return &plugin.ExecResponse{
        Result: &plugin.ExecResult{
            Payload: &pluginpb.PluginV1_ExecResult_Document{
                Document: &plugin.DocumentResult{Documents: docs},
            },
        },
    }, nil
}

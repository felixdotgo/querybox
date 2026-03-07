package main

import (
	"context"
	"sort"

	"github.com/felixdotgo/querybox/pkg/plugin"
	pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"
	"go.mongodb.org/mongo-driver/bson"
)

// GetCompletionFields samples up to 100 documents from the specified
// collection and returns the union of all top-level field names found.
func (m *mongoPlugin) GetCompletionFields(ctx context.Context, req *pluginpb.PluginV1_GetCompletionFieldsRequest) (*pluginpb.PluginV1_GetCompletionFieldsResponse, error) {
	client, defaultDB, err := connectMongo(ctx, req.Connection)
	if err != nil {
		return &pluginpb.PluginV1_GetCompletionFieldsResponse{}, nil
	}
	defer client.Disconnect(ctx)

	dbName := req.Database
	if dbName == "" {
		dbName = defaultDB
	}
	if dbName == "" {
		dbName = getDatabaseName(req.Connection)
	}

	collName := req.Collection
	if collName == "" {
		return &pluginpb.PluginV1_GetCompletionFieldsResponse{}, nil
	}

	coll := client.Database(dbName).Collection(collName)

	// $sample returns a random subset without a full collection scan.
	pipeline := bson.A{
		bson.D{{Key: "$sample", Value: bson.D{{Key: "size", Value: int32(100)}}}},
	}
	cursor, err := coll.Aggregate(ctx, pipeline)
	if err != nil {
		return &pluginpb.PluginV1_GetCompletionFieldsResponse{}, nil
	}
	defer cursor.Close(ctx)

	fieldSet := make(map[string]struct{})
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		for k := range doc {
			fieldSet[k] = struct{}{}
		}
	}

	fields := make([]*plugin.FieldInfo, 0, len(fieldSet))
	for name := range fieldSet {
		fields = append(fields, &plugin.FieldInfo{Name: name})
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].Name < fields[j].Name })

	return &pluginpb.PluginV1_GetCompletionFieldsResponse{Fields: fields}, nil
}

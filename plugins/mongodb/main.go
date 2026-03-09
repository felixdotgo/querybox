package main

import (
	"context"
	"fmt"
	"time"

	"github.com/felixdotgo/querybox/pkg/plugin"
	pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"
	"go.mongodb.org/mongo-driver/bson"
)

// mongoPlugin implements the protobuf PluginServiceServer interface for MongoDB.
type mongoPlugin struct {
    pluginpb.UnimplementedPluginServiceServer
}

func (m *mongoPlugin) Info(ctx context.Context, _ *pluginpb.PluginV1_InfoRequest) (*plugin.InfoResponse, error) {
    return &plugin.InfoResponse{
        Type:         plugin.TypeDriver,
        Name:         "MongoDB",
        Version:      "0.1.0",
        Description:  "MongoDB document database driver",
        Url:          "https://www.mongodb.com/",
        Author:       "MongoDB Inc.",
        Capabilities: []string{"query", "mutate-row"},
        Tags:         []string{"nosql", "document"},
        License:      "Apache-2.0",
        IconUrl:      "https://www.mongodb.com/assets/images/global/favicon.ico",
        Contact:      "https://www.mongodb.com/community/forums/",
    }, nil
}

func (m *mongoPlugin) DescribeSchema(ctx context.Context, req *plugin.DescribeSchemaRequest) (*plugin.DescribeSchemaResponse, error) {
    // MongoDB is schemaless; return empty response
    return &plugin.DescribeSchemaResponse{}, nil
}

func (m *mongoPlugin) AuthForms(ctx context.Context, _ *plugin.AuthFormsRequest) (*plugin.AuthFormsResponse, error) {
    basic := plugin.AuthForm{
        Key:  "basic",
        Name: "Basic",
        Fields: []*plugin.AuthField{
            {Type: plugin.AuthFieldText, Name: "host", Label: "Host", Required: true, Placeholder: "127.0.0.1", Value: "127.0.0.1"},
            {Type: plugin.AuthFieldNumber, Name: "port", Label: "Port", Placeholder: "27017", Value: "27017"},
            {Type: plugin.AuthFieldText, Name: "user", Label: "Username"},
            {Type: plugin.AuthFieldPassword, Name: "password", Label: "Password"},
            {Type: plugin.AuthFieldText, Name: "database", Label: "Database", Placeholder: "mydb"},
            {Type: plugin.AuthFieldText, Name: "auth_source", Label: "Auth Source", Placeholder: "admin", Value: "admin"},
            {Type: plugin.AuthFieldSelect, Name: "tls", Label: "TLS", Options: []string{"false", "true"}, Value: "false"},
        },
    }
    uri := plugin.AuthForm{
        Key:  "uri",
        Name: "URI",
        Fields: []*plugin.AuthField{
            {Type: plugin.AuthFieldText, Name: "uri", Label: "MongoDB URI", Required: true, Placeholder: "mongodb://user:pass@localhost:27017/mydb"},
        },
    }
    return &plugin.AuthFormsResponse{Forms: map[string]*plugin.AuthForm{"basic": &basic, "uri": &uri}}, nil
}

func (m *mongoPlugin) Exec(ctx context.Context, req *plugin.ExecRequest) (*plugin.ExecResponse, error) {
    client, dbname, err := connectMongo(ctx, req.Connection)
    if err != nil {
        return &plugin.ExecResponse{Error: fmt.Sprintf("connection error: %v", err)}, nil
    }
    defer client.Disconnect(ctx)

    if dbname == "" {
        dbname = getDatabaseName(req.Connection)
    }

    return execMQL(ctx, client.Database(dbname), req.Query)
}

func (m *mongoPlugin) ConnectionTree(ctx context.Context, req *plugin.ConnectionTreeRequest) (*plugin.ConnectionTreeResponse, error) {
    client, _, err := connectMongo(ctx, req.Connection)
    if err != nil {
        return &plugin.ConnectionTreeResponse{}, nil
    }
    defer client.Disconnect(ctx)

    dbResult, err := client.ListDatabases(ctx, bson.D{})
    if err != nil {
        return &plugin.ConnectionTreeResponse{}, nil
    }

    var dbNodes []*plugin.ConnectionTreeNode
    for _, dbInfo := range dbResult.Databases {
        dbName := dbInfo.Name
        db := client.Database(dbName)

        collNames, _ := db.ListCollectionNames(ctx, bson.D{})

        var collNodes []*plugin.ConnectionTreeNode
        for _, coll := range collNames {
            collNodes = append(collNodes, &plugin.ConnectionTreeNode{
                Key:      dbName + "." + coll,
                Label:    coll,
                NodeType: plugin.ConnectionTreeNodeTypeCollection,
                Actions: []*plugin.ConnectionTreeAction{
                    {
                        Type:   plugin.ConnectionTreeActionSelect,
                        Title:  "Find documents",
                        Query:  fmt.Sprintf("db.%s.find({})", coll),
                        Hidden: true,
                        NewTab: true,
                    },
                    {
                        Type:  plugin.ConnectionTreeActionDropTable,
                        Title: "Drop collection",
                        Query: fmt.Sprintf("db.%s.drop()", coll),
                    },
                },
            })
        }

        dbNodes = append(dbNodes, &plugin.ConnectionTreeNode{
            Key:      dbName,
            Label:    dbName,
            NodeType: plugin.ConnectionTreeNodeTypeDatabase,
            Children: collNodes,
            Actions: []*plugin.ConnectionTreeAction{
                {
                    Type:  plugin.ConnectionTreeActionCreateTable,
                    Title: "Create collection",
                    Query: `db.createCollection("new_collection")`,
                },
                {
                    Type:  plugin.ConnectionTreeActionDropDatabase,
                    Title: "Drop database",
                    Query: "db.dropDatabase()",
                },
            },
        })
    }

    return &plugin.ConnectionTreeResponse{Nodes: dbNodes}, nil
}

func (m *mongoPlugin) TestConnection(ctx context.Context, req *plugin.TestConnectionRequest) (*plugin.TestConnectionResponse, error) {
    timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()

    client, _, err := connectMongo(timeoutCtx, req.Connection)
    if err != nil {
        return &plugin.TestConnectionResponse{Ok: false, Message: fmt.Sprintf("connection error: %v", err)}, nil
    }
    defer client.Disconnect(timeoutCtx)

    if err := client.Ping(timeoutCtx, nil); err != nil {
        return &plugin.TestConnectionResponse{Ok: false, Message: fmt.Sprintf("ping error: %v", err)}, nil
    }
    return &plugin.TestConnectionResponse{Ok: true, Message: "Connection successful"}, nil
}


// MutateRow stub for MongoDB driver; always reports not supported.
func (m *mongoPlugin) MutateRow(ctx context.Context, req *plugin.MutateRowRequest) (*plugin.MutateRowResponse, error) {
    return &plugin.MutateRowResponse{Error: "not supported"}, nil
}

func main() {
    plugin.ServeCLI(&mongoPlugin{})
}

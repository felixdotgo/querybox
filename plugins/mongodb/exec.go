package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/felixdotgo/querybox/pkg/plugin"
	pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/protobuf/types/known/structpb"
)

// buildFindOptions builds MongoDB find options from parsed MQL args.
// Supported shell-style forms:
//   db.collection.find({filter})
//   db.collection.find({filter}, {projection})
//   db.collection.findOne({filter}, {projection})
// applyChainToFindOptions mutates the options based on chained methods.
func applyChainToFindOptions(opts *options.FindOptions, chain []chainOp) error {
    for _, c := range chain {
        switch strings.ToLower(c.Name) {
        case "sort":
            if c.Args == "" {
                continue
            }
            doc, err := parseBSONDoc(c.Args)
            if err != nil {
                return fmt.Errorf("sort parse error (args %q): %w", c.Args, err)
            }
            opts.SetSort(doc)
        case "limit":
            if c.Args == "" {
                continue
            }
            var v int64
            _, err := fmt.Sscan(c.Args, &v)
            if err != nil {
                return fmt.Errorf("limit parse error: %w", err)
            }
            opts.SetLimit(v)
        case "skip":
            if c.Args == "" {
                continue
            }
            var v int64
            _, err := fmt.Sscan(c.Args, &v)
            if err != nil {
                return fmt.Errorf("skip parse error: %w", err)
            }
            opts.SetSkip(v)
        case "batchsize":
            if c.Args == "" {
                continue
            }
            var v int32
            _, err := fmt.Sscan(c.Args, &v)
            if err != nil {
                return fmt.Errorf("batchSize parse error: %w", err)
            }
            opts.SetBatchSize(v)
        case "projection":
            if c.Args == "" {
                continue
            }
            doc, err := parseBSONDoc(c.Args)
            if err != nil {
                return fmt.Errorf("projection parse error: %w", err)
            }
            opts.SetProjection(doc)
        default:
            // other chained methods are ignored for now; they may be handled
            // when additional functionality is added or fall back to raw JSON.
        }
    }
    return nil
}

// buildFindOptions builds MongoDB find options from parsed MQL args and
// applies any chained operations.
func buildFindOptions(op string, args []string, chain []chainOp) (*options.FindOptions, error) {
    findOpts := options.Find()
    if op == "findOne" {
        findOpts.SetLimit(1)
    }

    if len(args) > 1 && strings.TrimSpace(args[1]) != "" {
        projection, err := parseBSONDoc(args[1])
        if err != nil {
            return nil, fmt.Errorf("projection parse error: %w", err)
        }
        findOpts.SetProjection(projection)
    }

    if err := applyChainToFindOptions(findOpts, chain); err != nil {
        return nil, err
    }

    return findOpts, nil
}

// execMQL executes a MongoDB shell-style query or a raw JSON command against db.
// runRawCommand attempts to execute a collection operation that we
// don't explicitly recognise.  It builds a simple command document where the
// operation name is the primary key and the collection name (if provided) is
// the value.  Additional arguments are merged when possible.
// buildCommandDoc constructs the command document that will be sent to
// `db.RunCommand` for an unrecognised operation.  The implementation mirrors
// the logic previously embedded in runRawCommand, but without performing any
// network calls; this makes it easier to unit‑test.
func buildCommandDoc(target, op string, args []string) (bson.D, error) {
    cmd := bson.D{{Key: op, Value: target}}
    for _, a := range args {
        if a == "" {
            continue
        }
        var doc bson.D
        if err := bson.UnmarshalExtJSON([]byte(a), false, &doc); err == nil {
            cmd = append(cmd, doc...)
            continue
        }
        var v interface{}
        if err := bson.UnmarshalExtJSON([]byte(a), false, &v); err == nil {
            cmd = append(cmd, bson.E{Key: fmt.Sprintf("arg%d", len(cmd)), Value: v})
            continue
        }
        cmd = append(cmd, bson.E{Key: fmt.Sprintf("arg%d", len(cmd)), Value: a})
    }
    return cmd, nil
}

func runRawCommand(ctx context.Context, db *mongo.Database, target, op string, args []string) (*plugin.ExecResponse, error) {
    cmd, err := buildCommandDoc(target, op, args)
    if err != nil {
        return &plugin.ExecResponse{Error: fmt.Sprintf("build command error: %v", err)}, nil
    }

    result := db.RunCommand(ctx, cmd)
    if result.Err() != nil {
        return &plugin.ExecResponse{Error: fmt.Sprintf("command error: %v", result.Err())}, nil
    }
    var raw bson.D
    if err := result.Decode(&raw); err != nil {
        return &plugin.ExecResponse{Error: fmt.Sprintf("decode error: %v", err)}, nil
    }
    s, err := bsonDocToStruct(raw)
    if err != nil {
        return &plugin.ExecResponse{Error: fmt.Sprintf("format error: %v", err)}, nil
    }
    return &plugin.ExecResponse{
        Result: &plugin.ExecResult{
            Payload: &pluginpb.PluginV1_ExecResult_Document{
                Document: &plugin.DocumentResult{Documents: []*structpb.Struct{s}},
            },
        },
    }, nil
}

func buildAggregateOptions(chain []chainOp) (*options.AggregateOptions, error) {
    aggrOpts := options.Aggregate()
    for _, c := range chain {
        switch strings.ToLower(c.Name) {
        case "allowdiskuse":
            if c.Args == "" {
                continue
            }
            var v bool
            _, err := fmt.Sscan(c.Args, &v)
            if err != nil {
                return nil, fmt.Errorf("allowDiskUse parse error: %w", err)
            }
            aggrOpts.SetAllowDiskUse(v)
        case "batchsize":
            if c.Args == "" {
                continue
            }
            var v int32
            _, err := fmt.Sscan(c.Args, &v)
            if err != nil {
                return nil, fmt.Errorf("batchSize parse error: %w", err)
            }
            aggrOpts.SetBatchSize(v)
        case "collation":
            if c.Args == "" {
                continue
            }
            col, err := parseBSONDoc(c.Args)
            if err != nil {
                return nil, fmt.Errorf("collation parse error: %w", err)
            }
            aggrOpts.SetCollation(&options.Collation{Locale: col.Map()["locale"].(string)})
        default:
            // ignore other chains for now
        }
    }
    return aggrOpts, nil
}

func execMQL(ctx context.Context, db *mongo.Database, query string) (*plugin.ExecResponse, error) {
    query = strings.TrimSpace(query)

    target, op, argsStr, chain, ok := parseMQLCommand(query)
    if ok {
        args := splitTopLevelArgs(argsStr)

        // Handle db-level operations (target is empty).
        if target == "" {
            switch op {
            case "dropDatabase":
                if err := db.Drop(ctx); err != nil {
                    return &plugin.ExecResponse{Error: fmt.Sprintf("dropDatabase error: %v", err)}, nil
                }
                return kvResponse(map[string]string{"result": "ok", "dropped": db.Name()}), nil

            case "createCollection":
                if len(args) == 0 {
                    return &plugin.ExecResponse{Error: "createCollection requires a collection name"}, nil
                }
                collName := strings.Trim(args[0], `"' `)
                if err := db.CreateCollection(ctx, collName); err != nil {
                    return &plugin.ExecResponse{Error: fmt.Sprintf("createCollection error: %v", err)}, nil
                }
                return kvResponse(map[string]string{"result": "ok", "created": collName}), nil

            case "listCollections", "getCollectionNames":
                names, err := db.ListCollectionNames(ctx, bson.D{})
                if err != nil {
                    return &plugin.ExecResponse{Error: fmt.Sprintf("listCollections error: %v", err)}, nil
                }
                result := make(map[string]string, len(names))
                for i, n := range names {
                    result[fmt.Sprintf("%d", i)] = n
                }
                return kvResponse(result), nil
            }
        }

        // Handle collection-level operations.
        coll := db.Collection(target)
        switch op {
        case "find", "findOne":
            filter := bson.D{}
            if len(args) > 0 && args[0] != "" {
                var err error
                filter, err = parseBSONDoc(args[0])
                if err != nil {
                    return &plugin.ExecResponse{Error: fmt.Sprintf("filter parse error: %v", err)}, nil
                }
            }
            findOpts, err := buildFindOptions(op, args, chain)
            if err != nil {
                return &plugin.ExecResponse{Error: err.Error()}, nil
            }
            cursor, err := coll.Find(ctx, filter, findOpts)
            if err != nil {
                return &plugin.ExecResponse{Error: fmt.Sprintf("find error: %v", err)}, nil
            }
            defer cursor.Close(ctx)
            return cursorToDocumentResponse(ctx, cursor)

        case "insertOne":
            if len(args) == 0 || args[0] == "" {
                return &plugin.ExecResponse{Error: "insertOne requires a document argument"}, nil
            }
            doc, err := parseBSONDoc(args[0])
            if err != nil {
                return &plugin.ExecResponse{Error: fmt.Sprintf("document parse error: %v", err)}, nil
            }
            res, err := coll.InsertOne(ctx, doc)
            if err != nil {
                return &plugin.ExecResponse{Error: fmt.Sprintf("insertOne error: %v", err)}, nil
            }
            return kvResponse(map[string]string{"insertedId": fmt.Sprintf("%v", res.InsertedID)}), nil

        case "insertMany":
            if len(args) == 0 || args[0] == "" {
                return &plugin.ExecResponse{Error: "insertMany requires a documents array argument"}, nil
            }
            arr, err := parseBSONArray(args[0])
            if err != nil {
                return &plugin.ExecResponse{Error: fmt.Sprintf("documents parse error: %v", err)}, nil
            }
            docs := make([]interface{}, len(arr))
            for i, v := range arr {
                docs[i] = v
            }
            res, err := coll.InsertMany(ctx, docs)
            if err != nil {
                return &plugin.ExecResponse{Error: fmt.Sprintf("insertMany error: %v", err)}, nil
            }
            ids := make([]string, len(res.InsertedIDs))
            for i, id := range res.InsertedIDs {
                ids[i] = fmt.Sprintf("%v", id)
            }
            return kvResponse(map[string]string{
                "insertedCount": fmt.Sprintf("%d", len(ids)),
                "insertedIds":   strings.Join(ids, ", "),
            }), nil

        case "updateOne", "updateMany", "replaceOne":
            if len(args) < 2 {
                return &plugin.ExecResponse{Error: fmt.Sprintf("%s requires filter and update arguments", op)}, nil
            }
            filter, err := parseBSONDoc(args[0])
            if err != nil {
                return &plugin.ExecResponse{Error: fmt.Sprintf("filter parse error: %v", err)}, nil
            }
            update, err := parseBSONDoc(args[1])
            if err != nil {
                return &plugin.ExecResponse{Error: fmt.Sprintf("update parse error: %v", err)}, nil
            }
            var matched, modified int64
            switch op {
            case "updateOne":
                res, err := coll.UpdateOne(ctx, filter, update)
                if err != nil {
                    return &plugin.ExecResponse{Error: fmt.Sprintf("updateOne error: %v", err)}, nil
                }
                matched, modified = res.MatchedCount, res.ModifiedCount
            case "updateMany":
                res, err := coll.UpdateMany(ctx, filter, update)
                if err != nil {
                    return &plugin.ExecResponse{Error: fmt.Sprintf("updateMany error: %v", err)}, nil
                }
                matched, modified = res.MatchedCount, res.ModifiedCount
            case "replaceOne":
                res, err := coll.ReplaceOne(ctx, filter, update)
                if err != nil {
                    return &plugin.ExecResponse{Error: fmt.Sprintf("replaceOne error: %v", err)}, nil
                }
                matched, modified = res.MatchedCount, res.ModifiedCount
            }
            return kvResponse(map[string]string{
                "matchedCount":  fmt.Sprintf("%d", matched),
                "modifiedCount": fmt.Sprintf("%d", modified),
            }), nil

        case "deleteOne", "deleteMany":
            filter := bson.D{}
            if len(args) > 0 && args[0] != "" {
                var err error
                filter, err = parseBSONDoc(args[0])
                if err != nil {
                    return &plugin.ExecResponse{Error: fmt.Sprintf("filter parse error: %v", err)}, nil
                }
            }
            var deleted int64
            if op == "deleteOne" {
                res, err := coll.DeleteOne(ctx, filter)
                if err != nil {
                    return &plugin.ExecResponse{Error: fmt.Sprintf("deleteOne error: %v", err)}, nil
                }
                deleted = res.DeletedCount
            } else {
                res, err := coll.DeleteMany(ctx, filter)
                if err != nil {
                    return &plugin.ExecResponse{Error: fmt.Sprintf("deleteMany error: %v", err)}, nil
                }
                deleted = res.DeletedCount
            }
            return kvResponse(map[string]string{"deletedCount": fmt.Sprintf("%d", deleted)}), nil

        case "aggregate":
            if len(args) == 0 || args[0] == "" {
                return &plugin.ExecResponse{Error: "aggregate requires a pipeline array argument"}, nil
            }
            pipeline, err := parseBSONArray(args[0])
            if err != nil {
                return &plugin.ExecResponse{Error: fmt.Sprintf("pipeline parse error: %v", err)}, nil
            }
            aggrOpts, _ := buildAggregateOptions(chain)
            cursor, err := coll.Aggregate(ctx, pipeline, aggrOpts)
            if err != nil {
                return &plugin.ExecResponse{Error: fmt.Sprintf("aggregate error: %v", err)}, nil
            }
            defer cursor.Close(ctx)
            return cursorToDocumentResponse(ctx, cursor)

        case "countDocuments":
            filter := bson.D{}
            if len(args) > 0 && args[0] != "" {
                var err error
                filter, err = parseBSONDoc(args[0])
                if err != nil {
                    return &plugin.ExecResponse{Error: fmt.Sprintf("filter parse error: %v", err)}, nil
                }
            }
            count, err := coll.CountDocuments(ctx, filter)
            if err != nil {
                return &plugin.ExecResponse{Error: fmt.Sprintf("countDocuments error: %v", err)}, nil
            }
            return kvResponse(map[string]string{"count": fmt.Sprintf("%d", count)}), nil

        case "estimatedDocumentCount":
            count, err := coll.EstimatedDocumentCount(ctx)
            if err != nil {
                return &plugin.ExecResponse{Error: fmt.Sprintf("estimatedDocumentCount error: %v", err)}, nil
            }
            return kvResponse(map[string]string{"count": fmt.Sprintf("%d", count)}), nil

        case "drop":
            if err := coll.Drop(ctx); err != nil {
                return &plugin.ExecResponse{Error: fmt.Sprintf("drop error: %v", err)}, nil
            }
            return kvResponse(map[string]string{"result": "ok", "dropped": target}), nil

        case "createIndex":
            if len(args) == 0 || args[0] == "" {
                return &plugin.ExecResponse{Error: "createIndex requires an index keys document"}, nil
            }
            keys, err := parseBSONDoc(args[0])
            if err != nil {
                return &plugin.ExecResponse{Error: fmt.Sprintf("keys parse error: %v", err)}, nil
            }
            indexModel := mongo.IndexModel{Keys: keys}
            name, err := coll.Indexes().CreateOne(ctx, indexModel)
            if err != nil {
                return &plugin.ExecResponse{Error: fmt.Sprintf("createIndex error: %v", err)}, nil
            }
            return kvResponse(map[string]string{"name": name}), nil

        case "distinct":
            if len(args) == 0 || args[0] == "" {
                return &plugin.ExecResponse{Error: "distinct requires a field name"}, nil
            }
            field := strings.Trim(args[0], `"' `)
            filter := bson.D{}
            if len(args) > 1 && args[1] != "" {
                var err error
                filter, err = parseBSONDoc(args[1])
                if err != nil {
                    return &plugin.ExecResponse{Error: fmt.Sprintf("filter parse error: %v", err)}, nil
                }
            }
            values, err := coll.Distinct(ctx, field, filter)
            if err != nil {
                return &plugin.ExecResponse{Error: fmt.Sprintf("distinct error: %v", err)}, nil
            }
            strs := make([]string, len(values))
            for i, v := range values {
                strs[i] = fmt.Sprintf("%v", v)
            }
            return kvResponse(map[string]string{
                "values": strings.Join(strs, ", "),
                "count":  fmt.Sprintf("%d", len(values)),
            }), nil
        }

        // unhandled op – try the generic command fallback before failing
        return runRawCommand(ctx, db, target, op, args)
    }

    // Fall back to a raw JSON command document.
    if strings.HasPrefix(query, "{") {
        var cmd bson.D
        if err := bson.UnmarshalExtJSON([]byte(query), false, &cmd); err != nil {
            return &plugin.ExecResponse{Error: fmt.Sprintf("invalid command JSON: %v", err)}, nil
        }
        result := db.RunCommand(ctx, cmd)
        if result.Err() != nil {
            return &plugin.ExecResponse{Error: fmt.Sprintf("command error: %v", result.Err())}, nil
        }
        var raw bson.D
        if err := result.Decode(&raw); err != nil {
            return &plugin.ExecResponse{Error: fmt.Sprintf("decode error: %v", err)}, nil
        }
        s, err := bsonDocToStruct(raw)
        if err != nil {
            return &plugin.ExecResponse{Error: fmt.Sprintf("format error: %v", err)}, nil
        }
        return &plugin.ExecResponse{
            Result: &plugin.ExecResult{
                Payload: &pluginpb.PluginV1_ExecResult_Document{
                    Document: &plugin.DocumentResult{Documents: []*structpb.Struct{s}},
                },
            },
        }, nil
    }

    return &plugin.ExecResponse{
        Error: "unsupported query format\n" +
            "Examples:\n" +
            "  db.users.find({})\n" +
            "  db.users.insertOne({\"name\": \"Alice\"})\n" +
            "  db.users.updateOne({\"name\": \"Alice\"}, {\"$set\": {\"age\": 30}})\n" +
            "  db.users.deleteOne({\"name\": \"Alice\"})\n" +
            "  db.users.aggregate([{\"$group\": {\"_id\": \"$status\"}}])\n" +
            "  {\"ping\": 1}",
    }, nil
}

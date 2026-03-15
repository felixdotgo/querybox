package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/felixdotgo/querybox/pkg/certs"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// credentialPayload is the JSON shape stored in connection["credential_blob"].
type credentialPayload struct {
    Form   string            `json:"form"`
    Values map[string]string `json:"values"`
}

// buildURI constructs a MongoDB connection URI from the connection map.
// Returns the URI string, the explicitly configured database name, and any error.
func buildURI(connection map[string]string) (string, string, error) {
    // Direct URI key takes precedence.
    if u, ok := connection["uri"]; ok && u != "" {
        return u, "", nil
    }

    if blob, ok := connection["credential_blob"]; ok && blob != "" {
        var payload credentialPayload
        if err := json.Unmarshal([]byte(blob), &payload); err != nil {
            return "", "", fmt.Errorf("invalid credential blob: %w", err)
        }
        if u, ok := payload.Values["uri"]; ok && u != "" {
            return u, "", nil
        }
        return buildURIFromValues(payload.Values)
    }

    return buildURIFromValues(connection)
}

// buildURIFromValues constructs a MongoDB URI from a flat key/value map.
func buildURIFromValues(values map[string]string) (string, string, error) {
    host := values["host"]
    if host == "" {
        host = "127.0.0.1"
    }
    port := values["port"]
    if port == "" {
        port = "27017"
    }
    user := values["user"]
    pass := values["password"]
    dbname := values["database"]
    authSource := values["auth_source"]
    if authSource == "" {
        authSource = "admin"
    }
    tlsMode := values["tls"]

    u := url.URL{
        Scheme: "mongodb",
        Host:   fmt.Sprintf("%s:%s", host, port),
    }
    if user != "" {
        u.User = url.UserPassword(user, pass)
    }
    if dbname != "" {
        u.Path = "/" + dbname
    }
    q := url.Values{}
    if user != "" {
        q.Set("authSource", authSource)
    }
    if tlsMode == "true" {
        q.Set("tls", "true")
    }
    if len(q) > 0 {
        u.RawQuery = q.Encode()
    }
    return u.String(), dbname, nil
}

// getDatabaseName returns the database name from the connection map, if specified.
func getDatabaseName(connection map[string]string) string {
    if blob, ok := connection["credential_blob"]; ok && blob != "" {
        var payload credentialPayload
        if json.Unmarshal([]byte(blob), &payload) == nil {
            if d := payload.Values["database"]; d != "" {
                return d
            }
        }
    }
    return connection["database"]
}

// connectMongo builds a *mongo.Client from the connection map.
// The caller is responsible for calling client.Disconnect.
func connectMongo(ctx context.Context, connection map[string]string) (*mongo.Client, string, error) {
    uri, dbname, err := buildURI(connection)
    if err != nil {
        return nil, "", err
    }
    if uri == "" {
        return nil, "", fmt.Errorf("missing connection parameters")
    }

    opts := options.Client().ApplyURI(uri)

    // Attach embedded root CA pool when TLS is requested.
    if strings.Contains(uri, "tls=true") || strings.Contains(uri, "ssl=true") {
        if pool, e := certs.RootCertPool(); e == nil {
            opts.SetTLSConfig(&tls.Config{RootCAs: pool})
        }
    }

    client, err := mongo.Connect(ctx, opts)
    if err != nil {
        return nil, "", fmt.Errorf("connect error: %w", err)
    }
    return client, dbname, nil
}

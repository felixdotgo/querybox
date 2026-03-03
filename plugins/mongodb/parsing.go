package main

import (
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
)

// parseBSONDoc parses a JSON / relaxed extended JSON string into a bson.D.
func parseBSONDoc(s string) (bson.D, error) {
    s = strings.TrimSpace(s)
    if s == "" || s == "{}" {
        return bson.D{}, nil
    }
    var doc bson.D
    if err := bson.UnmarshalExtJSON([]byte(s), false, &doc); err != nil {
        return bson.D{}, fmt.Errorf("invalid JSON document: %w", err)
    }
    return doc, nil
}

// parseBSONArray parses a JSON array string into a bson.A.
func parseBSONArray(s string) (bson.A, error) {
    s = strings.TrimSpace(s)
    if s == "" || s == "[]" {
        return bson.A{}, nil
    }
    var arr bson.A
    if err := bson.UnmarshalExtJSON([]byte(s), false, &arr); err != nil {
        return bson.A{}, fmt.Errorf("invalid JSON array: %w", err)
    }
    return arr, nil
}

// splitTopLevelArgs splits a string by commas that are not nested inside
// brackets or string literals. This allows parsing multi-argument function
// calls such as `{filter}, {update}` or `[pipeline], {}`.
func splitTopLevelArgs(s string) []string {
    var args []string
    depth := 0
    inStr := false
    strChar := rune(0)
    escape := false
    start := 0

    for i, r := range s {
        if escape {
            escape = false
            continue
        }
        if r == '\\' && inStr {
            escape = true
            continue
        }
        if !inStr && (r == '"' || r == '\'') {
            inStr = true
            strChar = r
            continue
        }
        if inStr && r == strChar {
            inStr = false
            continue
        }
        if inStr {
            continue
        }
        switch r {
        case '{', '[', '(':
            depth++
        case '}', ']', ')':
            depth--
        case ',':
            if depth == 0 {
                args = append(args, strings.TrimSpace(s[start:i]))
                start = i + 1
            }
        }
    }
    if tail := strings.TrimSpace(s[start:]); tail != "" {
        args = append(args, tail)
    }
    return args
}

// parseMQLCommand parses a MongoDB shell-style query such as:
//
//	db.collection.find({...})
//	db.createCollection("name")
//
// It returns the target (collection name for collection ops, empty for db-level
// ops), the operation name, the raw argument string, and an ok flag.
func parseMQLCommand(query string) (target, op, argsStr string, ok bool) {
    query = strings.TrimSpace(query)
    if !strings.HasPrefix(query, "db.") {
        return
    }
    rest := query[3:] // strip "db."

    // Find the first opening parenthesis – everything before it is "target.op".
    parenIdx := strings.IndexByte(rest, '(')
    if parenIdx < 0 {
        return
    }
    funcPart := rest[:parenIdx]

    lastDot := strings.LastIndex(funcPart, ".")
    if lastDot < 0 {
        // No dot → top-level db operation, e.g. db.dropDatabase()
        target = ""
        op = strings.TrimSpace(funcPart)
    } else {
        target = strings.TrimSpace(funcPart[:lastDot])
        op = strings.TrimSpace(funcPart[lastDot+1:])
    }

    // Extract the content inside the outermost parentheses (balanced).
    inner := rest[parenIdx+1:]
    depth := 1
    strInner := false
    strInnerChar := rune(0)
    escInner := false

    for i, r := range inner {
        if escInner {
            escInner = false
            continue
        }
        if r == '\\' && strInner {
            escInner = true
            continue
        }
        if !strInner && (r == '"' || r == '\'') {
            strInner = true
            strInnerChar = r
            continue
        }
        if strInner && r == strInnerChar {
            strInner = false
            continue
        }
        if strInner {
            continue
        }
        switch r {
        case '(', '[', '{':
            depth++
        case ')', ']', '}':
            depth--
            if depth == 0 {
                argsStr = strings.TrimSpace(inner[:i])
                ok = true
                return
            }
        }
    }
    return
}

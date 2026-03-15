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

// chainOp represents a single method call following the primary
// collection operation (e.g. ".sort({a:1})" or ".limit(10)").
type chainOp struct {
    Name string
    Args string
}

// parseMQLCommand parses a MongoDB shell-style query such as:
//
//	db.collection.find({...}).sort({a:1}).limit(10)
//	db.createCollection("name")
//
// It returns the target (collection name for collection ops, empty for
// db-level ops), the operation name, the raw argument string, any chained
// operations, and an ok flag.
func parseMQLCommand(query string) (target, op, argsStr string, chain []chainOp, ok bool) {
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

    // Extract argument string for the primary operation.
    inner := rest[parenIdx+1:]
    depth := 1
    strInner := false
    strInnerChar := rune(0)
    escInner := false
    endIdx := -1

    // scan until the matching closing parenthesis for the first call
scanLoop:
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
                endIdx = i + 1 // position just after closing paren
                ok = true
                break scanLoop
            }
        }
    }
    if !ok {
        return
    }

    // parse any chained method calls following the first invocation
    rest = strings.TrimSpace(inner[endIdx:])
    for len(rest) > 0 {
        if !strings.HasPrefix(rest, ".") {
            break
        }
        rest = rest[1:]
        // method name up to next '('
        nameEnd := strings.IndexAny(rest, "( ")
        if nameEnd < 0 {
            break
        }
        name := strings.TrimSpace(rest[:nameEnd])
        rest = rest[nameEnd:]
        // expect arguments in parentheses
        if !strings.HasPrefix(rest, "(") {
            break
        }
        // find matching closing parenthesis
        depth = 1
        strInner = false
        strInnerChar = rune(0)
        escInner = false
        argStart := 1
        argEnd := -1
        for i := 1; i < len(rest); i++ {
            r := rune(rest[i])
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
                    argEnd = i
                    // break out of the outer for loop
                    goto chainEnd
                }
            }
        }
    chainEnd:
        if argEnd < 0 {
            break
        }
        args := strings.TrimSpace(rest[argStart:argEnd])
        chain = append(chain, chainOp{Name: name, Args: args})
        rest = strings.TrimSpace(rest[argEnd+1:])
    }
    return
}

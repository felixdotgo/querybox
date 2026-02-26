# Feature: Event System

## Contract

- **Backend is the sole event producer.** Go services emit events via `app.Event.Emit`. Frontend never calls `Events.Emit` for domain topics.
- **Frontend is a pure consumer.** Components subscribe via `Events.On` and react — no re-fetch RPC needed when the event payload is sufficient.
- **Events emit after successful DB write** — never speculatively.
- **Event constants** declared in `services/events.go` as Go `const` values. TypeScript listeners use the same string literals.

---

## Event Catalogue

| Event | Emitted by | Payload | When |
|-------|-----------|---------|------|
| `app:log` | All services | `LogEntry{Level, Message, Timestamp}` | Every significant service action |
| `connection:created` | `ConnectionService.CreateConnection` | `ConnectionCreatedEvent{Connection}` | After successful DB insert |
| `connection:deleted` | `ConnectionService.DeleteConnection` | `ConnectionDeletedEvent{ID}` | After successful DB delete |

`app:log` is a **stream channel**, not a state-change event — it does not follow the past-tense verb rule.

---

## Naming Rules

### Backend event names: `<domain>:<past-tense-verb>`

| Rule | Correct | Wrong |
|------|---------|-------|
| Lowercase only | `connection:created` | `Connection:Created` |
| Colon separator | `plugin:scanned` | `plugin.scanned` |
| Domain is a singular noun | `connection:deleted` | `connections:deleted` |
| Verb is past tense for state changes | `connection:created` | `connection:create` |
| Declared as Go `const` | ✓ | Inline string literals |

### Log messages: `"<MethodName>: <lowercase description>"`

| Rule | Correct | Wrong |
|------|---------|-------|
| Method name is PascalCase | `"CreateConnection: ..."` | `"create_connection: ..."` |
| Description starts lowercase | `"CreateConnection: creating 'db1'"` | `"CreateConnection: Creating 'db1'"` |
| No trailing period | `"connection deleted"` | `"connection deleted."` |
| Single-quoted identifiers | `"creating 'my-db'"` | `"creating my-db"` |
| KV context in parentheses | `"(driver: mysql, id: abc)"` | `"[driver=mysql]"` |

**Lifecycle templates:**
```go
// start
"CreateConnection: creating 'db1' (driver: mysql)"
// success
"CreateConnection: 'db1' created successfully (id: abc)"
"ListConnections: found 3 connection(s)"
// error
"CreateConnection: failed to store credential for 'db1': <err>"
"GetCredential: connection 'xyz' not found: <err>"
```

**Log levels:**

| Level | When |
|-------|------|
| `info` | Normal lifecycle: start, success, counts |
| `warn` | Recoverable: fallback triggered, optional resource missing |
| `error` | Non-recoverable: DB failure, credential loss, plugin crash |

### Vue component emits: `kebab-case`

| Rule | Correct | Wrong |
|------|---------|-------|
| Kebab-case | `"tab-closed"` | `"tabClosed"` |
| Past tense for notifications | `"connection-selected"` | `"select-connection"` |
| Noun phrase for data delivery | `"query-result"` | `"send-query-result"` |
| Imperative only for parent requests | `"toggle-logs"` | — |
| Never use backend event names | ✓ | `emit("connection:created")` |

---

## Adding a New Event

1. Add `const EventXxx = "domain:verb"` in `services/events.go` with a doc comment.
2. Add payload struct next to the constant (if new).
3. Add `emitXxx` helper following the nil-safe pattern in `services/events.go`.
4. Call helper **after** DB write succeeds.
5. Update the Event Catalogue table above.
6. Add TypeScript listener in the frontend — never emit from the frontend.

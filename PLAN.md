# HTTP Agent Context (HAC) - Pitch Document

## The Problem

AI agents are becoming the primary consumers of HTTP APIs. Today, when an agent calls `DELETE /users/123`, it has no way to know from the API itself that this action is irreversible, affects associated data, or requires the user to have no active subscriptions. A human developer knows this because they read the docs once and hardcoded that understanding into a client. An agent discovers and reasons about APIs at runtime - it needs this context delivered inline.

The existing solutions don't address this:

- **MCP** is a separate protocol layer. You build MCP servers that wrap your API. Powerful, but heavy. Most APIs will never get MCP servers.
- **OpenAPI** describes the technical contract but lacks runtime agent guidance - no reversibility, blast radius, cost, or recovery hints.
- **HATEOAS specs** (HAL, Siren, JSON:API) focus on resource navigation and relationships, not agent reasoning context.
- **A2A** solves agent-to-agent communication, not API consumption.

**The gap: there is no HTTP-native standard for APIs to communicate agent-relevant context within normal HTTP responses.**

## The Proposal

**HTTP Agent Context (HAC)** is a lightweight, backwards-compatible extension to HTTP APIs that provides AI agents with the context they need to reason about API actions safely and effectively.

It works through HTTP's existing content negotiation:

1. Agent requests enriched responses via `Accept: application/vnd.hac+json`
2. Server responds with normal data wrapped in an envelope that includes agent context metadata
3. Normal clients never request this content type and are completely unaffected

### Example

**Normal API call:**
```
GET /users/123
Accept: application/json
```
```json
{
  "id": 123,
  "name": "Alice",
  "email": "alice@example.com",
  "status": "active"
}
```

**Agent API call:**
```
GET /users/123
Accept: application/vnd.hac+json
```
```json
{
  "data": {
    "id": 123,
    "name": "Alice",
    "email": "alice@example.com",
    "status": "active"
  },
  "_hac": {
    "description": "An active user account.",
    "actions": [
      {
        "rel": "update",
        "method": "PATCH",
        "href": "/users/123",
        "description": "Update user profile fields. Changes are saved immediately and visible to the user.",
        "safety": {
          "mutability": "reversible",
          "blast_radius": "self"
        },
        "fields": [
          {"name": "name", "type": "string", "description": "Display name"},
          {"name": "email", "type": "string", "description": "Primary email. Changing this triggers a verification email."}
        ]
      },
      {
        "rel": "deactivate",
        "method": "POST",
        "href": "/users/123/deactivate",
        "description": "Deactivate this account. The user loses access immediately. Can be reactivated within 30 days, after which the account is permanently deleted.",
        "safety": {
          "mutability": "reversible",
          "reversible_within": "30d",
          "blast_radius": "self",
          "confirmation_recommended": true
        }
      },
      {
        "rel": "delete",
        "method": "DELETE",
        "href": "/users/123",
        "description": "Permanently delete this user and all associated data (orders, preferences, activity). Cannot be undone. Fails with 409 if user has active subscriptions.",
        "safety": {
          "mutability": "irreversible",
          "blast_radius": "self_and_associated",
          "confirmation_recommended": true
        },
        "preconditions": ["User must have no active subscriptions"]
      }
    ],
    "related": [
      {"rel": "orders", "href": "/users/123/orders", "description": "Orders placed by this user", "count": 17},
      {"rel": "subscriptions", "href": "/users/123/subscriptions", "description": "Active subscriptions"}
    ]
  }
}
```

### Agent-Enriched Errors

When agents encounter errors, HAC provides recovery guidance:

**Normal error:**
```json
{"error": "Cannot delete user with active subscriptions"}
```

**HAC error:**
```json
{
  "error": {
    "code": "active_subscriptions",
    "message": "Cannot delete user with active subscriptions",
    "retryable": false,
    "recovery": {
      "description": "Cancel all active subscriptions before deleting the user.",
      "action": {
        "method": "GET",
        "href": "/users/123/subscriptions?status=active",
        "description": "List active subscriptions that must be cancelled first"
      }
    }
  }
}
```

### Discovery

```
GET /
Accept: application/vnd.hac+json
```
```json
{
  "_hac": {
    "name": "Acme API",
    "version": "2.1",
    "description": "Acme Corp's customer and order management API.",
    "resources": [
      {
        "rel": "users",
        "href": "/users",
        "description": "User accounts. Supports listing, creation, update, and deletion.",
        "methods": ["GET", "POST"]
      },
      {
        "rel": "orders",
        "href": "/orders",
        "description": "Customer orders. Read-only for most consumers. Order creation requires 'orders:write' scope.",
        "methods": ["GET", "POST"]
      }
    ]
  }
}
```

## How It Differs from Existing Standards

| Standard | What it does | HAC's relationship |
|----------|-------------|-------------------|
| **MCP** | Separate protocol for agent-tool integration | HAC is HTTP-native. Complementary - HAC for APIs that won't build MCP servers |
| **OpenAPI** | Describes API contract at design time | HAC provides runtime context in responses. Can auto-generate HAC from OpenAPI |
| **HAL** | Hypermedia links in responses | HAC adds agent reasoning context (safety, descriptions, recovery), not just links |
| **Siren** | Hypermedia links + actions | Closest relative. HAC adds safety metadata, LLM-optimized descriptions, error recovery |
| **JSON:API** | Resource relationship envelope | Different envelope purpose: JSON:API is about data relationships, HAC is about agent guidance |
| **A2A** | Agent-to-agent communication | Different layer entirely |

## Key Design Decisions

### 1. Content negotiation, not headers
Using `Accept: application/vnd.hac+json` rather than a custom header because:
- It's how HTTP was designed to handle this
- Caches and CDNs understand `Vary: Accept`
- Server can cleanly return `406` if unsupported
- No new header registration needed

### 2. Descriptions written for LLMs
The single most impactful aspect of HAC. API descriptions today are written for human developers. HAC descriptions are written for LLM consumption: explicit about consequences, constraints, and edge cases.

Human developer description: *"Deletes a user resource"*

HAC description: *"Permanently delete this user and all associated data (orders, preferences, activity). Cannot be undone. Fails with 409 if user has active subscriptions."*

### 3. Safety metadata is first-class
Every action carries a `safety` object:
- `mutability`: `read_only` | `reversible` | `irreversible`
- `blast_radius`: `self` | `self_and_associated` | `many` | `all`
- `reversible_within`: duration string (e.g., "30d") - optional
- `confirmation_recommended`: boolean - hint that agents should verify with the user
- `cost`: optional object for actions with financial implications

### 4. Error recovery is structured
Errors include `recovery` objects that tell the agent what to do next - not just what went wrong.

### 5. Incremental adoption
A server can add HAC to one endpoint at a time. No all-or-nothing migration.

## The Tooling Opportunity

Once the spec exists, the tooling practically builds itself:

### For API Providers
**HAC middleware** (Go, Node, Python): Drop into an existing API server. Auto-generates HAC metadata from:
- HTTP method semantics (GET = read_only, DELETE = likely irreversible)
- OpenAPI spec annotations (if present)
- Route patterns (/users/{id} = single resource)
- Developer-provided annotations for the rest

### For Agent Developers
**HAC client library**: Consumes HAC responses and provides:
- Capability discovery ("what can I do with this resource?")
- Safety checking ("is this action safe to perform without user confirmation?")
- Error recovery ("what should I do about this error?")
- Action building ("construct a valid request for this action")

## Why Now

- MCP adoption is growing but only ~10% of developers use it regularly. Most APIs will remain plain HTTP.
- 80%+ of APIs lack agent-optimized metadata.
- AI agents are shifting from demos to production - they need to reason about API safety reliably.
- HTTP content negotiation is a proven, zero-cost adoption mechanism.
- The window to establish a standard is open before the ecosystem fragments further.

## Next Steps

1. Formalize this into an RFC-style specification
2. Build reference middleware implementation (Go)
3. Build reference agent client library
4. Publish spec and solicit feedback from API and AI communities


# HTTP Agent Context (HAC) Specification

**Version:** 1.0-draft
**Status:** Draft
**Date:** 2025-06

## Table of Contents

1. [Introduction](#1-introduction)
2. [Content Negotiation](#2-content-negotiation)
3. [HAC Response Envelope](#3-hac-response-envelope)
4. [Actions](#4-actions)
5. [Safety Metadata](#5-safety-metadata)
6. [Error Responses](#6-error-responses)
7. [API Discovery](#7-api-discovery)
8. [Extensibility](#8-extensibility)
9. [Security Considerations](#9-security-considerations)
10. [IANA Considerations](#10-iana-considerations)
11. [Relationship to Other Specifications](#11-relationship-to-other-specifications)
12. [Implementation Guidance](#12-implementation-guidance-non-normative)
13. [Appendices](#13-appendices)

---

## 1. Introduction

### 1.1 Purpose

HTTP Agent Context (HAC) is a lightweight, backwards-compatible extension to HTTP APIs that provides AI agents with the context they need to reason about API actions safely and effectively.

AI agents increasingly consume HTTP APIs at runtime, discovering capabilities and reasoning about actions dynamically rather than relying on hardcoded client logic. Existing API description formats (OpenAPI, HAL, Siren) were designed for human developers or mechanical client code, not for LLM-based agents that need to assess safety, understand consequences, and recover from errors autonomously.

HAC addresses this gap by defining a JSON envelope that wraps normal API payloads with agent-oriented metadata: LLM-optimized descriptions, safety classifications, available actions, error recovery guidance, and resource discovery.

### 1.2 Goals

- Provide AI agents with the runtime context needed to reason about API actions safely.
- Use HTTP's existing content negotiation mechanism so adoption is incremental and non-breaking.
- Remain simple enough that a single endpoint can adopt HAC without any other infrastructure changes.
- Define machine-readable safety metadata (mutability, blast radius, cost) as first-class concepts.
- Provide structured error recovery so agents can self-correct.

### 1.3 Non-Goals

- Replacing MCP, OpenAPI, or any existing specification.
- Defining authentication or authorization mechanisms.
- Specifying pagination, filtering, or sorting conventions.
- Defining agent-to-agent communication protocols.

### 1.4 Conventions

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD", "SHOULD NOT", "RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### 1.5 Terminology

| Term | Definition |
|------|-----------|
| **Agent** | An AI system (typically LLM-based) that consumes HTTP APIs at runtime. |
| **HAC envelope** | The top-level JSON object containing `data` and `_hac` in a success response. |
| **HAC metadata** | The `_hac` object within a response. |
| **Action** | A hypermedia control describing an available operation the agent can invoke. |
| **Safety metadata** | Structured classification of an action's risk characteristics. |
| **Recovery** | Structured guidance attached to an error response that tells the agent how to resolve the error. |

---

## 2. Content Negotiation

### 2.1 Media Type

HAC defines the media type:

```
application/vnd.hac+json
```

### 2.2 Request Behavior

An agent that wants HAC-enriched responses MUST include the HAC media type in the `Accept` request header:

```http
GET /users/123 HTTP/1.1
Accept: application/vnd.hac+json
```

An agent MAY include multiple media types with quality factors to indicate fallback preferences:

```http
Accept: application/vnd.hac+json, application/json;q=0.9
```

### 2.3 Response Behavior

A server that supports HAC for the requested resource MUST:

1. Return the response with `Content-Type: application/vnd.hac+json`.
2. Include the `Vary: Accept` response header to ensure correct cache behavior.
3. Return the response body conforming to the HAC envelope structure defined in [Section 3](#3-hac-response-envelope).

A server that does not support HAC for the requested resource SHOULD return `406 Not Acceptable` if the HAC media type is the only acceptable type. If the client provided fallback types, the server SHOULD respond with the best available alternative.

### 2.4 Versioning

The HAC specification version is communicated within the response body via the `_hac.version` field (see [Section 3.2](#32-the-_hac-object)), not through the media type.

Future versions of this specification MAY define a `version` media type parameter (e.g., `application/vnd.hac+json; version=2`) if breaking changes require media-type-level negotiation.

### 2.5 Caching

Servers MUST include `Vary: Accept` in HAC responses so that caches distinguish between HAC and non-HAC representations of the same resource. Standard HTTP caching semantics (`Cache-Control`, `ETag`, `Last-Modified`) apply to HAC responses as they do to any HTTP response.

---

## 3. HAC Response Envelope

### 3.1 Top-Level Structure

A HAC success response is a JSON object with exactly two top-level keys:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `data` | any | REQUIRED | The original API payload. May be any valid JSON value. |
| `_hac` | object | REQUIRED | Agent-oriented metadata. See [Section 3.2](#32-the-_hac-object). |

The `data` field contains the API's normal response payload unchanged. This MAY be a JSON object, array, string, number, boolean, or null.

Servers MUST NOT alter the structure of the original payload when wrapping it in the HAC envelope. An agent that strips the `_hac` key and extracts `data` MUST receive the exact payload that a non-HAC request would have returned.

### 3.2 The `_hac` Object

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `version` | string | REQUIRED | The HAC specification version (e.g., `"1.0"`). |
| `description` | string | OPTIONAL | An LLM-optimized description of the resource. |
| `actions` | array | OPTIONAL | Available actions. See [Section 4](#4-actions). |
| `related` | array | OPTIONAL | Related resources. See [Section 3.3](#33-related-resources). |

#### 3.2.1 Description Authoring Guidelines

The `description` field (and all `description` fields throughout HAC) SHOULD be written for LLM consumption. Effective descriptions:

- State what the resource or action **is** and what it **does**, not just its name.
- Are explicit about **consequences** (especially destructive or costly ones).
- Mention **constraints** and **preconditions** that affect whether the action will succeed.
- Note **edge cases** and **side effects** (e.g., "Changing email triggers a verification email").
- Use plain, direct language. Avoid jargon that assumes domain knowledge the agent may not have.

**Example -- human-developer description:**
> Deletes a user resource.

**Example -- HAC LLM-optimized description:**
> Permanently delete this user and all associated data (orders, preferences, activity). Cannot be undone. Fails with 409 if user has active subscriptions.

### 3.3 Related Resources

Each entry in the `related` array is an object:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `rel` | string | REQUIRED | Link relation type. |
| `href` | string | REQUIRED | URI or URI Template for the related resource. |
| `description` | string | OPTIONAL | LLM-optimized description of the related resource. |

Servers MAY include additional properties on related resource objects (e.g., `count`). Clients MUST ignore unrecognized properties.

---

## 4. Actions

### 4.1 Action Object

Each entry in the `actions` array is an object representing a hypermedia control:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `rel` | string | REQUIRED | Link relation type. |
| `method` | string | REQUIRED | HTTP method (`GET`, `HEAD`, `POST`, `PUT`, `PATCH`, `DELETE`, `OPTIONS`). |
| `href` | string | REQUIRED | URI or URI Template ([RFC 6570](https://www.rfc-editor.org/rfc/rfc6570)) for the action. |
| `description` | string | OPTIONAL | LLM-optimized description. See [Section 3.2.1](#321-description-authoring-guidelines). |
| `safety` | object | OPTIONAL | Safety metadata. See [Section 5](#5-safety-metadata). |
| `fields` | array | OPTIONAL | Input fields. See [Section 4.3](#43-fields). |
| `preconditions` | array | OPTIONAL | Human-readable strings describing preconditions. |

### 4.2 Link Relations

Action `rel` values SHOULD use [IANA-registered link relation types](https://www.iana.org/assignments/link-relations/link-relations.xhtml) where an appropriate type exists (e.g., `edit`, `collection`, `next`).

When no IANA type is appropriate, implementations SHOULD use either:

- A **URI** that can be dereferenced for documentation (e.g., `https://api.example.com/rels/deactivate`).
- A **descriptive extension token** using lowercase letters, digits, and hyphens (e.g., `deactivate`, `cancel-subscription`).

### 4.3 Fields

Each entry in the `fields` array describes an input parameter:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | REQUIRED | Field name as expected in the request. |
| `type` | string | REQUIRED | JSON Schema type: `string`, `number`, `integer`, `boolean`, `array`, `object`. |
| `description` | string | OPTIONAL | LLM-optimized description, including constraints and side effects. |
| `required` | boolean | OPTIONAL | Whether the field is required. Defaults to `false`. |
| `enum` | array | OPTIONAL | Allowed values. |
| `default` | any | OPTIONAL | Default value if not provided. |

### 4.4 URI Templates

When an action `href` contains a URI Template ([RFC 6570](https://www.rfc-editor.org/rfc/rfc6570)), the template variables SHOULD correspond to `fields` entries so the agent can construct a valid request.

---

## 5. Safety Metadata

The `safety` object on an action provides structured risk assessment metadata. All fields are OPTIONAL, but servers are RECOMMENDED to include at least `mutability` and `blast_radius` for any action that mutates state.

### 5.1 `mutability`

Indicates whether the action mutates state and whether the mutation is reversible.

| Value | Meaning |
|-------|---------|
| `read_only` | The action does not modify any state. |
| `reversible` | The action modifies state, but the change can be undone. |
| `irreversible` | The action modifies state and the change cannot be undone. |

**Agent guidance:** An agent SHOULD freely invoke `read_only` actions. For `irreversible` actions, the agent SHOULD confirm with the user before proceeding unless the user has explicitly delegated authority.

### 5.2 `blast_radius`

Indicates the scope of resources affected by the action.

| Value | Meaning |
|-------|---------|
| `self` | Only the targeted resource is affected. |
| `self_and_associated` | The targeted resource and its directly associated resources are affected. |
| `many` | Multiple resources beyond the target are affected. |
| `all` | All resources in the system (or a major subsystem) are affected. |

**Agent guidance:** Agents SHOULD apply increasing caution as blast radius increases. Actions with `many` or `all` blast radius SHOULD always be confirmed with the user.

### 5.3 `reversible_within`

An [ISO 8601 duration](https://en.wikipedia.org/wiki/ISO_8601#Durations) string (e.g., `P30D` for 30 days, `PT1H` for one hour) indicating the window during which a `reversible` action can be undone.

This field is only meaningful when `mutability` is `reversible`. Servers SHOULD omit it for `read_only` and `irreversible` actions.

**Agent guidance:** An agent SHOULD communicate the reversibility window to the user when confirming the action (e.g., "This can be undone within 30 days").

### 5.4 `confirmation_recommended`

A boolean (default `false`) indicating that the server recommends the agent confirm with the user before invoking this action.

This is a hint, not a hard requirement. An agent MAY proceed without confirmation if the user has explicitly authorized the class of action.

### 5.5 `cost`

An object describing financial cost associated with the action:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `amount` | number | REQUIRED | The monetary cost. |
| `currency` | string | REQUIRED | ISO 4217 currency code (e.g., `"USD"`). |
| `description` | string | OPTIONAL | Human-readable explanation of the cost. |

**Agent guidance:** An agent MUST NOT invoke a cost-bearing action without user confirmation unless the user has explicitly pre-authorized spending up to a specified limit.

---

## 6. Error Responses

### 6.1 Error Envelope

A HAC error response is a JSON object with a single top-level key:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `error` | object | REQUIRED | Structured error information. |

The `error` object contains:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `code` | string | REQUIRED | Machine-readable error code. |
| `message` | string | REQUIRED | Human- and LLM-readable error message. |
| `retryable` | boolean | OPTIONAL | Whether the request may be retried. Defaults to `false`. |
| `retry_after` | integer | OPTIONAL | Seconds to wait before retrying. |
| `recovery` | object | OPTIONAL | Recovery guidance. See [Section 6.2](#62-recovery). |

A HAC error response MUST NOT contain a `data` key. The absence of `data` and presence of `error` distinguishes error responses from success responses.

### 6.2 Recovery

The `recovery` object provides structured guidance for resolving the error:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `description` | string | REQUIRED | LLM-optimized description of what the agent should do to resolve the error. |
| `actions` | array | OPTIONAL | Concrete actions the agent can take to recover. Each entry is an Action object as defined in [Section 4.1](#41-action-object). |

### 6.3 HTTP Status Codes

HAC does not redefine HTTP status code semantics. Servers SHOULD use standard status codes:

- **4xx** for client errors (invalid request, authorization failure, precondition violation).
- **5xx** for server errors.
- **409 Conflict** when a precondition is not met (e.g., resource has dependencies that prevent deletion).
- **429 Too Many Requests** with `retryable: true` and an appropriate `retry_after` value.

---

## 7. API Discovery

### 7.1 Root Endpoint Convention

A HAC-enabled API SHOULD respond to a HAC-negotiated request at its root URL (`/`) with a discovery document:

```http
GET / HTTP/1.1
Accept: application/vnd.hac+json
```

### 7.2 Discovery Response Structure

The discovery response is a JSON object with a single top-level key:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `_hac` | object | REQUIRED | Discovery metadata. |

The `_hac` object in a discovery response contains:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | REQUIRED | Human-readable name of the API. |
| `version` | string | OPTIONAL | The API version (not the HAC spec version). |
| `description` | string | OPTIONAL | LLM-optimized description of the API. |
| `resources` | array | REQUIRED | Available top-level resources. |

Each entry in `resources` is:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `rel` | string | REQUIRED | Link relation type identifying the resource. |
| `href` | string | REQUIRED | Base URI for the resource. |
| `description` | string | OPTIONAL | LLM-optimized description. |
| `methods` | array | OPTIONAL | HTTP methods supported at this endpoint. |

---

## 8. Extensibility

### 8.1 Unknown Fields

Clients MUST ignore unrecognized fields in HAC responses. This ensures forward compatibility as the specification evolves and allows servers to include implementation-specific metadata.

### 8.2 Vendor Extensions

Servers MAY include vendor-specific fields in any HAC object. Vendor extension field names SHOULD be prefixed with `x-` to avoid collisions with future specification-defined fields (e.g., `x-acme-internal-id`).

### 8.3 Profile Parameter

Servers MAY use a `profile` media type parameter to indicate additional constraints or extensions:

```
Content-Type: application/vnd.hac+json; profile="https://api.example.com/hac-profile/v2"
```

The `profile` URI SHOULD be dereferenceable and SHOULD describe the additional constraints or extensions in use.

---

## 9. Security Considerations

### 9.1 Information Disclosure

HAC metadata (actions, descriptions, related resources) may reveal API capabilities to unauthorized parties. Servers SHOULD only include actions and metadata that the authenticated user is authorized to access. Servers MUST NOT expose actions that the current user cannot perform due to authorization constraints.

### 9.2 Action Href Validation

Agents MUST validate that action `href` values belong to the expected API origin before invoking them. An `href` that redirects to a third-party domain could be used for server-side request forgery (SSRF) or credential theft.

Agents SHOULD:
- Reject absolute URIs with an origin different from the API's origin unless the agent has been explicitly configured to trust the target.
- Resolve relative URIs against the API's base URL.

### 9.3 Prompt Injection via Descriptions

HAC `description` fields are intended for LLM consumption. A compromised or malicious server could craft descriptions that attempt to manipulate agent behavior (prompt injection).

Agents SHOULD:
- Treat all description content as untrusted data.
- Apply appropriate sanitization or sandboxing before incorporating descriptions into LLM prompts.
- Never execute code or follow instructions embedded in description fields.

### 9.4 Transport Security

HAC responses SHOULD only be served over HTTPS. Serving HAC over plain HTTP risks interception and modification of metadata that agents rely on for safety decisions.

---

## 10. IANA Considerations

### 10.1 Media Type Registration

This specification registers the following media type:

| Field | Value |
|-------|-------|
| Type name | application |
| Subtype name | vnd.hac+json |
| Required parameters | None |
| Optional parameters | `version`, `profile` |
| Encoding considerations | Same as `application/json` ([RFC 8259](https://www.rfc-editor.org/rfc/rfc8259)) |
| Security considerations | See [Section 9](#9-security-considerations) |
| Interoperability considerations | None |
| Published specification | This document |
| Fragment identifier considerations | Same as `application/json` |

---

## 11. Relationship to Other Specifications

### 11.1 OpenAPI

[OpenAPI](https://www.openapis.org/) describes API contracts at design time. HAC provides runtime context within responses. The two are complementary: servers MAY auto-generate HAC metadata from OpenAPI annotations, and OpenAPI specifications can document that an API supports HAC content negotiation.

### 11.2 HAL

[HAL (Hypertext Application Language)](https://stateless.group/hal_specification.html) defines `_links` and `_embedded` for hypermedia navigation. HAC differs in focus: while HAL describes resource relationships, HAC adds agent reasoning context -- safety metadata, LLM-optimized descriptions, and error recovery. A server could theoretically support both HAL and HAC representations via content negotiation.

### 11.3 Siren

[Siren](https://github.com/kevinswiber/siren) is the closest relative to HAC, providing both links and actions with field definitions. HAC extends this concept with safety metadata, cost information, preconditions, and recovery guidance specifically designed for AI agent consumption.

### 11.4 JSON:API

[JSON:API](https://jsonapi.org/) defines a complete envelope for resource relationships, pagination, and sparse fieldsets. HAC's envelope serves a different purpose: agent guidance rather than data relationship management. The two could coexist if a server wrapped JSON:API-structured data inside HAC's `data` field.

### 11.5 MCP (Model Context Protocol)

[MCP](https://modelcontextprotocol.io/) is a separate protocol layer where servers expose tools, resources, and prompts to AI agents through a dedicated transport. HAC is HTTP-native and works within existing API infrastructure. They are complementary: HAC serves the long tail of APIs that will never build MCP servers.

### 11.6 A2A (Agent-to-Agent)

[A2A](https://github.com/google/A2A) defines communication between AI agents. HAC defines communication between an API and an agent. They operate at different layers and do not overlap.

---

## 12. Implementation Guidance (Non-Normative)

### 12.1 Incremental Adoption

HAC is designed for incremental adoption. A server can:

1. Start with a single endpoint -- add HAC support to the most agent-relevant resource.
2. Include only `version` and `description` initially -- even without actions or safety metadata, descriptions alone add significant value.
3. Add actions, safety metadata, and discovery over time as the investment proves its value.

### 12.2 Auto-Generation from OpenAPI

Servers with existing OpenAPI specifications can bootstrap HAC metadata:

- Map `GET` operations to `mutability: "read_only"`.
- Map `DELETE` operations to `mutability: "irreversible"` (as a conservative default).
- Map `PUT`/`PATCH` operations to `mutability: "reversible"`.
- Extract field definitions from request body schemas.
- Use operation `summary` and `description` as starting points for HAC descriptions (but rewrite for LLM consumption).

### 12.3 Agent Decision Framework

Agents consuming HAC responses SHOULD use the following decision framework:

1. **Read the description** to understand the resource and available actions.
2. **Check safety metadata** before invoking any mutating action.
3. **Confirm with the user** when `confirmation_recommended` is `true`, when `mutability` is `irreversible`, when `blast_radius` is `many` or `all`, or when `cost` is present.
4. **Check preconditions** before invoking an action and resolve them if possible.
5. **On error, check recovery** guidance and follow the suggested actions.

---

## 13. Appendices

### Appendix A: Complete Success Response Example

```http
GET /users/123 HTTP/1.1
Accept: application/vnd.hac+json
```

```http
HTTP/1.1 200 OK
Content-Type: application/vnd.hac+json
Vary: Accept
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
    "version": "1.0",
    "description": "An active user account.",
    "actions": [
      {
        "rel": "edit",
        "method": "PATCH",
        "href": "/users/123",
        "description": "Update user profile fields. Changes are saved immediately and visible to the user.",
        "safety": {
          "mutability": "reversible",
          "blast_radius": "self"
        },
        "fields": [
          {
            "name": "name",
            "type": "string",
            "description": "Display name"
          },
          {
            "name": "email",
            "type": "string",
            "description": "Primary email. Changing this triggers a verification email."
          }
        ]
      },
      {
        "rel": "deactivate",
        "method": "POST",
        "href": "/users/123/deactivate",
        "description": "Deactivate this account. The user loses access immediately. Can be reactivated within 30 days, after which the account is permanently deleted.",
        "safety": {
          "mutability": "reversible",
          "reversible_within": "P30D",
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
        "preconditions": [
          "User must have no active subscriptions"
        ]
      }
    ],
    "related": [
      {
        "rel": "orders",
        "href": "/users/123/orders",
        "description": "Orders placed by this user",
        "count": 17
      },
      {
        "rel": "subscriptions",
        "href": "/users/123/subscriptions",
        "description": "Active subscriptions"
      }
    ]
  }
}
```

### Appendix B: Complete Error Response Example

```http
DELETE /users/123 HTTP/1.1
Accept: application/vnd.hac+json
```

```http
HTTP/1.1 409 Conflict
Content-Type: application/vnd.hac+json
Vary: Accept
```

```json
{
  "error": {
    "code": "active_subscriptions",
    "message": "Cannot delete user with active subscriptions.",
    "retryable": false,
    "recovery": {
      "description": "Cancel all active subscriptions before deleting the user.",
      "actions": [
        {
          "rel": "related",
          "method": "GET",
          "href": "/users/123/subscriptions?status=active",
          "description": "List active subscriptions that must be cancelled first."
        }
      ]
    }
  }
}
```

### Appendix C: Complete Discovery Response Example

```http
GET / HTTP/1.1
Accept: application/vnd.hac+json
```

```http
HTTP/1.1 200 OK
Content-Type: application/vnd.hac+json
Vary: Accept
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

### Appendix D: Cost-Bearing Action Example

```json
{
  "rel": "upgrade",
  "method": "POST",
  "href": "/users/123/subscription/upgrade",
  "description": "Upgrade the user's subscription to the Pro plan. Billing begins immediately and is prorated for the current period.",
  "safety": {
    "mutability": "reversible",
    "reversible_within": "P14D",
    "blast_radius": "self",
    "confirmation_recommended": true,
    "cost": {
      "amount": 29.99,
      "currency": "USD",
      "description": "Monthly Pro plan subscription (prorated)"
    }
  },
  "fields": [
    {
      "name": "plan",
      "type": "string",
      "required": true,
      "enum": ["pro", "enterprise"],
      "description": "The target plan to upgrade to."
    }
  ]
}
```

### Appendix E: JSON Schema References

The following JSON Schemas formally define the structures described in this specification:

- **[hac-envelope.schema.json](schema/hac-envelope.schema.json)** -- HAC success response envelope ([Section 3](#3-hac-response-envelope))
- **[hac-error.schema.json](schema/hac-error.schema.json)** -- HAC error response envelope ([Section 6](#6-error-responses))
- **[hac-discovery.schema.json](schema/hac-discovery.schema.json)** -- HAC discovery response ([Section 7](#7-api-discovery))

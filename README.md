# HTTP Agent Context (HAC)

**A lightweight, backwards-compatible extension to HTTP APIs that provides AI agents with the context they need to reason about API actions safely and effectively.**

HAC works through HTTP content negotiation: agents request `Accept: application/vnd.hac+json` and receive normal API data wrapped in an envelope containing safety metadata, available actions, error recovery guidance, and LLM-optimized descriptions. Non-agent clients are completely unaffected.

## Specification

- **[HAC Specification v1](spec/hac-spec-v1.md)** -- the full RFC-style specification

## JSON Schemas

- [`hac-envelope.schema.json`](spec/schema/hac-envelope.schema.json) -- HAC success response envelope
- [`hac-error.schema.json`](spec/schema/hac-error.schema.json) -- HAC error response envelope
- [`hac-discovery.schema.json`](spec/schema/hac-discovery.schema.json) -- HAC root discovery response

## Quick Example

```
GET /users/123
Accept: application/vnd.hac+json
```

```json
{
  "data": {
    "id": 123,
    "name": "Alice",
    "email": "alice@example.com"
  },
  "_hac": {
    "version": "1.0",
    "description": "An active user account.",
    "actions": [
      {
        "rel": "delete",
        "method": "DELETE",
        "href": "/users/123",
        "description": "Permanently delete this user and all associated data. Cannot be undone.",
        "safety": {
          "mutability": "irreversible",
          "blast_radius": "self_and_associated",
          "confirmation_recommended": true
        }
      }
    ]
  }
}
```

## Status

This specification is a **draft**. Feedback is welcome via issues and pull requests.

## License

This specification is made available under the [Creative Commons Attribution 4.0 International License (CC BY 4.0)](https://creativecommons.org/licenses/by/4.0/).

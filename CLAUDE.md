# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

HAC (HTTP Agent Context) is a **specification repository** — not a code implementation. It defines a lightweight, backwards-compatible HTTP extension that provides AI agents with runtime context for safe API consumption via content negotiation (`Accept: application/vnd.hac+json`).

**Status**: DRAFT specification
**License**: CC BY 4.0

## Repository Structure

- `spec/hac-spec-v1.md` — The full RFC-style specification (primary artifact)
- `spec/schema/` — JSON Schema (Draft 2020-12) definitions:
  - `hac-envelope.schema.json` — Success response envelope (`{data, _hac}`)
  - `hac-error.schema.json` — Error response with recovery guidance
  - `hac-discovery.schema.json` — API root discovery response
- `PLAN.md` — Design rationale, problem statement, and comparison to related technologies (MCP, OpenAPI, HAL, Siren, JSON:API, A2A)
- `README.md` — Project introduction and quick example

## Commands

Schema validation (configured but not yet wired up):
```bash
npx ajv-cli validate -s spec/schema/hac-envelope.schema.json -d <example.json>
```

## Key Concepts

- **Content Negotiation**: Agents request HAC via `Accept: application/vnd.hac+json`; non-agent clients are unaffected
- **Envelope Pattern**: Wraps existing API `data` with `_hac` metadata (actions, safety, descriptions)
- **Safety Metadata**: `mutability` (read_only|reversible|irreversible), `blast_radius` (self|self_and_associated|many|all), `reversible_within`, `confirmation_recommended`, `cost`
- **Recovery Guidance**: Error responses include structured recovery actions agents can execute
- **Incremental Adoption**: Individual endpoints can adopt HAC without infrastructure changes

## Writing Conventions

- Specification text follows RFC 2119 keyword conventions (MUST, SHOULD, MAY)
- All `description` fields in schemas and spec examples should be **LLM-optimized**: explicit about consequences, constraints, and edge cases rather than terse human shorthand
- JSON schemas use Draft 2020-12 (`https://json-schema.org/draft/2020-12/schema`)

# Changelog

All notable changes to this project are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Because this is a client library, "breaking" means a change that stops
existing code compiling or changes what a call does. A new exported
field or method is additive, thus not breaking.

While this project is at 0.x the exported API is not yet frozen:
breaking changes bump the minor version and are listed under
`### Changed`, with a note on what to do about them. From 1.0.0 onward
they will require a new major version.

## [Unreleased]

## [0.1.0] - 2026-09-02

First public release. The API is complete and verified against a live
Firezone portal, but ships as 0.x while it gets real use. See the
versioning note above for what that means for compatibility.

### Added

- Read-write services for Sites, Resources, Policies, Groups, Actors and
  Client devices, with Gateways nested under Sites, memberships nested
  under Groups, and pool members nested under Resources.
- Read-only services for the five auth provider types and the three
  directory connection types.
- Cursor pagination on every list method (`ListOptions` → `Page[T]`).
- Typed errors: `APIError` carrying RFC 9457 problem details, with
  `IsNotFound`, `IsValidation`, `IsRateLimited`, `IsForbidden`,
  `IsUnauthorized` and `IsConflict` predicates.
- Automatic retry with exponential backoff and jitter on HTTP 429,
  honouring `Retry-After`. Configurable via `WithRetry` and
  `WithRetryMaxWait`.
- `Null[T]` with `Set` and `Clear`, so nullable fields on merge-patch
  update requests can be cleared as well as set.
- `Version`, sent as part of the default `User-Agent`.

### Notes

- `Update` methods send `PATCH`. The API routes `PATCH` and `PUT` to the
  same handler, but a partial update is what `PATCH` means, and sending
  the matching verb insures against the two ever diverging.
  `Memberships.ReplaceAll` and `PoolMembers.ReplaceAll` send `PUT`,
  where the API genuinely distinguishes them, as do `Verify` and
  `Unverify`.
- Some endpoints are deliberately not wrapped; the README lists which,
  and the `GatewaysService` doc comment explains the Gateway token ones
  in particular.

[Unreleased]: https://github.com/firezone/firezone-go/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/firezone/firezone-go/releases/tag/v0.1.0

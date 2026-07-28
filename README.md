# firezone (api-client)

A Go client for the Firezone REST API.

```go
import firezone "github.com/firezone/firezone-go"

client, err := firezone.NewClient("https://api.firezone.dev", token)
if err != nil {
	// invalid base URL
}

site, err := client.Sites.Create(ctx, &firezone.CreateSiteRequest{Name: "primary-dc"})
if err != nil {
	if firezone.IsConflict(err) {
		// a site with that name already exists
	}
	return err
}

gw, err := client.Sites.Gateways(site.ID).Provision(ctx, &firezone.ProvisionGatewayRequest{
	Name: "gw-nyc-1",
})
// gw.Token is only ever returned here, on Provision - the API never
// re-exposes it. See the Gateway type's doc comment before storing it
// anywhere long-lived.
```

`baseURL` passed to `NewClient` is always the bare API host
(`https://api.firezone.dev`).

## Resources

`Client` exposes one service per resource:
* `Sites`
* `Resources`
* `Policies`
* `Groups`
* `Actors`
* `Gateways` nested under `Sites` (`client.Sites.Gateways(siteID)`)

This matches the API's own URL nesting.

Every list method takes `*ListOptions{Limit, PageCursor}` and returns a
`*Page[T]{Data, Metadata}`. See the resource file for each type's exact
fields (`sites.go`, `resources.go`, `policies.go`, `groups.go`,
`memberships.go`, `actors.go`, `gateways.go`).

## Errors

Non-2xx responses are parsed into a typed `*APIError` (RFC 9457
problem+json). Use the `Is*` predicates rather than checking status
codes directly:

```go
switch {
case firezone.IsNotFound(err):
case firezone.IsConflict(err):
case firezone.IsValidation(err):
	// err.(*firezone.APIError).ValidationErrors has field-level detail
case firezone.IsRateLimited(err):
case firezone.IsForbidden(err):
case firezone.IsUnauthorized(err):
}
```

## Retries

Requests are retried automatically on HTTP 429 with exponential
backoff, honoring the API's `Retry-After` header (5 attempts by
default). Disable or tune this via `firezone.WithRetry`:

```go
client, _ := firezone.NewClient(endpoint, token, firezone.WithRetry(false, 0))
```

## Testing

```bash
mise run test              # unit tests, no server needed (httptest-based)
mise run test-acceptance   # requires FIREZONE_ENDPOINT/FIREZONE_TOKEN
```

To get a token for the acceptance tests, boot `mix phx.server` in a
`firezone/firezone` checkout and run that repo's
`elixir/script/seed_api_client_token.exs` - see the
[`terraform-provider-firezone`](https://github.com/firezone/terraform-provider-firezone)
README's "Local development" section for the full sequence.

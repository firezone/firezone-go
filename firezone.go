// Package firezone provides a hand-written Go client for the Firezone
// REST API (https://www.firezone.dev).
//
// Construct a [Client] with [NewClient], then call methods on its
// resource services (Sites, Resources, Policies, Groups, Actors, and
// Gateways nested under Sites):
//
//	client, err := firezone.NewClient("https://api.firezone.dev", token)
//	site, err := client.Sites.Create(ctx, &firezone.CreateSiteRequest{Name: "primary-dc"})
//
// The API is currently unversioned - baseURL is the bare API host, with
// no path prefix of any kind. (URL path versioning was tried and rolled
// back before shipping; if it returns, it'll live in exactly one place
// here rather than every call site.)
package firezone

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
)

// requestBody is a request body captured as raw bytes rather than an
// io.Reader, so requestWithRetry can construct a fresh reader for each
// retry attempt - an io.Reader can only be drained once, which would
// silently send an empty body on any attempt after the first.
type requestBody []byte

func (b requestBody) reader() io.Reader {
	if b == nil {
		return nil
	}
	return bytes.NewReader(b)
}

const defaultUserAgent = "firezone-go-client"

// String returns a pointer to s. Useful for optional string fields
// (e.g. [GroupListOptions.DirectoryID]) where a plain string's zero
// value can't distinguish "not set" from "set to the empty string".
func String(s string) *string {
	return &s
}

// Client is a Firezone REST API client.
type Client struct {
	baseURL    *url.URL
	token      string
	httpClient *http.Client
	userAgent  string

	retryEnabled bool
	maxRetries   int

	// Sites manages Sites and, nested under them, Gateways.
	Sites *SitesService
	// Resources manages Resources.
	Resources *ResourcesService
	// Policies manages Policies.
	Policies *PoliciesService
	// Groups manages Groups and, nested under them, memberships.
	Groups *GroupsService
	// Actors manages Actors.
	Actors *ActorsService
	// ClientDevices manages Client devices. Named for [ClientDevice],
	// since Clients is too easily confused with this type itself.
	ClientDevices *ClientsService
	// EntraDirectories reads Microsoft Entra directory connections
	// (read-only).
	EntraDirectories *EntraDirectoriesService
	// GoogleDirectories reads Google Workspace directory connections
	// (read-only).
	GoogleDirectories *GoogleDirectoriesService
	// OktaDirectories reads Okta directory connections (read-only).
	OktaDirectories *OktaDirectoriesService
}

// Option configures a [Client].
type Option func(*Client)

// WithHTTPClient sets the underlying *http.Client used for requests.
// The default is http.DefaultClient.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// WithUserAgent sets the User-Agent header sent with every request.
func WithUserAgent(ua string) Option {
	return func(c *Client) { c.userAgent = ua }
}

// defaultMaxRetries is the retry budget a client gets unless
// [WithRetry] says otherwise.
//
// The API rate limits per account with a token bucket: 20 requests of
// burst, refilling at roughly one per second. A Terraform apply or
// destroy at the default parallelism of 10 drains that burst almost
// immediately and then proceeds at the refill rate, so a request can
// legitimately need to wait out several seconds of queue ahead of it.
// The budget is sized for that, not for a transient blip.
const defaultMaxRetries = 8

// WithRetry configures automatic retry-with-backoff on HTTP 429
// (rate limited) responses. Retries are enabled by default with a
// budget of defaultMaxRetries.
//
// Waits honor the response's Retry-After header when present, falling
// back to exponential backoff, and always add jitter so concurrent
// callers don't retry in lockstep.
func WithRetry(enabled bool, maxRetries int) Option {
	return func(c *Client) {
		c.retryEnabled = enabled
		c.maxRetries = maxRetries
	}
}

// NewClient constructs a Firezone API client. baseURL is the bare API
// host (e.g. "https://api.firezone.dev") - do not include a version
// segment. token is the Bearer token for an api_client actor.
func NewClient(baseURL, token string, opts ...Option) (*Client, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("firezone: invalid base URL: %w", err)
	}

	c := &Client{
		baseURL:      parsed,
		token:        token,
		httpClient:   http.DefaultClient,
		userAgent:    defaultUserAgent,
		retryEnabled: true,
		maxRetries:   defaultMaxRetries,
	}

	for _, opt := range opts {
		opt(c)
	}

	c.Sites = &SitesService{client: c}
	c.Resources = &ResourcesService{client: c}
	c.Policies = &PoliciesService{client: c}
	c.Groups = &GroupsService{client: c}
	c.Actors = &ActorsService{client: c}
	c.ClientDevices = &ClientsService{client: c}
	c.EntraDirectories = &EntraDirectoriesService{client: c}
	c.GoogleDirectories = &GoogleDirectoriesService{client: c}
	c.OktaDirectories = &OktaDirectoriesService{client: c}

	return c, nil
}

// wrapBody marshals v as JSON, nested under key, matching the API's
// request body shape (e.g. {"site": {"name": "..."}}). A nil v produces
// no body at all.
func wrapBody(key string, v any) (requestBody, error) {
	if v == nil {
		return nil, nil
	}
	body, err := json.Marshal(map[string]any{key: v})
	if err != nil {
		return nil, fmt.Errorf("firezone: encoding request body: %w", err)
	}
	return requestBody(body), nil
}

// dataEnvelope mirrors the {"data": ...} shape every non-list API
// response is wrapped in.
type dataEnvelope[T any] struct {
	Data T `json:"data"`
}

// listEnvelope mirrors the {"data": [...], "metadata": {...}} shape
// every list API response is wrapped in.
type listEnvelope[T any] struct {
	Data     []T              `json:"data"`
	Metadata pageMetadataBody `json:"metadata"`
}

// rawRequest performs a single HTTP round trip against the API, with no
// retry logic - callers needing retry-on-429 behavior use requestWithRetry.
// The returned body is the raw response bytes; a non-2xx status yields a
// non-nil *APIError.
func (c *Client) rawRequest(ctx context.Context, method, requestPath string, query url.Values, body requestBody) ([]byte, error) {
	u := *c.baseURL
	u.Path = path.Join(u.Path, requestPath)
	if query != nil {
		u.RawQuery = query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body.reader())
	if err != nil {
		return nil, fmt.Errorf("firezone: building request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("firezone: performing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("firezone: reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, parseAPIError(resp, respBody)
	}

	return respBody, nil
}

// do performs a request and, on success, unmarshals the response's
// "data" field into out. out may be nil for responses with no body to
// capture (e.g. DELETE).
func (c *Client) do(ctx context.Context, method, requestPath string, query url.Values, body requestBody, out any) error {
	respBody, err := c.requestWithRetry(ctx, method, requestPath, query, body)
	if err != nil {
		return err
	}
	if out == nil || len(respBody) == 0 {
		return nil
	}

	var env dataEnvelope[json.RawMessage]
	if err := json.Unmarshal(respBody, &env); err != nil {
		return fmt.Errorf("firezone: decoding response: %w", err)
	}
	if err := json.Unmarshal(env.Data, out); err != nil {
		return fmt.Errorf("firezone: decoding response data: %w", err)
	}
	return nil
}

// doList performs a request and unmarshals the response's "data" and
// "metadata" fields into a Page[T]. It's a package-level generic
// function (not a Client method) because Go methods can't introduce
// their own type parameters.
func doList[T any](ctx context.Context, c *Client, method, requestPath string, query url.Values) (*Page[T], error) {
	respBody, err := c.requestWithRetry(ctx, method, requestPath, query, nil)
	if err != nil {
		return nil, err
	}

	var env listEnvelope[T]
	if err := json.Unmarshal(respBody, &env); err != nil {
		return nil, fmt.Errorf("firezone: decoding list response: %w", err)
	}

	return &Page[T]{Data: env.Data, Metadata: env.Metadata.toMetadata()}, nil
}

// listOptionsToQuery converts ListOptions into the API's query
// parameters (limit, page_cursor). Always returns a non-nil (possibly
// empty) url.Values, so callers can safely add further query
// parameters to the result.
func listOptionsToQuery(opts *ListOptions) url.Values {
	q := url.Values{}
	if opts == nil {
		return q
	}
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.PageCursor != "" {
		q.Set("page_cursor", opts.PageCursor)
	}
	return q
}

// filterQuery builds the query for a List method: opts' pagination
// parameters plus zero or more "key=value" filters, each set only when
// its value is non-empty. Every List method with resource-specific
// filters funnels through this so filter-building stays consistent.
func filterQuery(opts ListOptions, filters ...[2]string) url.Values {
	q := listOptionsToQuery(&opts)
	for _, kv := range filters {
		if kv[1] != "" {
			q.Set(kv[0], kv[1])
		}
	}
	return q
}

// Package catalog provides an operatorbundle.CatalogClient backed by an operator catalog
// GraphQL API (by default the Red Hat Ecosystem Catalog,
// https://catalog.redhat.com/api/containers/graphql/), using the find_operator_bundles query.
package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/operatorbundle"
)

// DefaultURL is the Red Hat Ecosystem Catalog GraphQL endpoint.
const DefaultURL = "https://catalog.redhat.com/api/containers/graphql/"

// maxErrBodyLen bounds how much of an error response body is echoed in error messages.
const maxErrBodyLen = 500

// Client queries an operator catalog GraphQL API.
type Client struct {
	url        string
	token      string
	httpClient *http.Client
}

// NewClient builds a catalog Client. An empty url falls back to DefaultURL; an empty token
// means unauthenticated requests.
func NewClient(url, token string) *Client {
	if url == "" {
		url = DefaultURL
	}
	return &Client{
		url:        url,
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// bundleData mirrors the find_operator_bundles data records we consume.
type bundleData struct {
	Package         string `json:"package"`
	ChannelName     string `json:"channel_name"`
	Version         string `json:"version"`
	VersionOriginal string `json:"version_original"`
	CSVName         string `json:"csv_name"`
	CSVDisplayName  string `json:"csv_display_name"`
	CreationDate    string `json:"creation_date"`
	RelatedImages   []struct {
		Image  string `json:"image"`
		Digest string `json:"digest"`
		Name   string `json:"name"`
	} `json:"related_images"`
}

type graphqlResponse struct {
	Data struct {
		FindOperatorBundles struct {
			Data []bundleData `json:"data"`
		} `json:"find_operator_bundles"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// FindInstalledBundle returns the bundle shipping the image with the given digest.
func (c *Client) FindInstalledBundle(ctx context.Context, digest string) (*operatorbundle.Bundle, error) {
	query := fmt.Sprintf(`{ find_operator_bundles(filter: {related_images: {digest: {eq: %s}}}, page_size: 1) `+
		`{ data { package channel_name version version_original csv_name csv_display_name creation_date `+
		`related_images { image digest name } } } }`, jsonString(digest))

	bundles, err := c.query(ctx, query)
	if err != nil {
		return nil, err
	}
	if len(bundles) == 0 {
		return nil, nil
	}
	b := toBundle(bundles[0])
	return &b, nil
}

// FindCandidateBundles returns bundles of the given package/channel created on or after
// sinceCreationDate, newest first.
func (c *Client) FindCandidateBundles(ctx context.Context, pkg, channel, sinceCreationDate string) ([]operatorbundle.Bundle, error) {
	query := fmt.Sprintf(`{ find_operator_bundles(filter: {and: [{package: {eq: %s}}, {creation_date: {ge: %s}}, {channel_name: {eq: %s}}]}, `+
		`page_size: 50, sort_by: [{field: "creation_date", order: DESC}]) `+
		`{ data { package channel_name version version_original csv_name csv_display_name creation_date `+
		`related_images { image digest name } } } }`,
		jsonString(pkg), jsonString(sinceCreationDate), jsonString(channel))

	bundles, err := c.query(ctx, query)
	if err != nil {
		return nil, err
	}
	result := make([]operatorbundle.Bundle, 0, len(bundles))
	for _, b := range bundles {
		result = append(result, toBundle(b))
	}
	return result, nil
}

func (c *Client) query(ctx context.Context, query string) ([]bundleData, error) {
	body, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		return nil, errors.Wrap(err, "marshaling GraphQL request")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(err, "building GraphQL request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "executing GraphQL request")
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "reading GraphQL response")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("catalog returned HTTP %d: %s", resp.StatusCode, truncate(string(respBody), maxErrBodyLen))
	}

	var parsed graphqlResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, errors.Wrap(err, "decoding GraphQL response")
	}
	if len(parsed.Errors) > 0 {
		msgs := make([]string, 0, len(parsed.Errors))
		for _, e := range parsed.Errors {
			msgs = append(msgs, e.Message)
		}
		return nil, errors.Errorf("catalog GraphQL errors: %s", strings.Join(msgs, "; "))
	}
	return parsed.Data.FindOperatorBundles.Data, nil
}

func toBundle(b bundleData) operatorbundle.Bundle {
	images := make([]operatorbundle.RelatedImage, 0, len(b.RelatedImages))
	for _, ri := range b.RelatedImages {
		images = append(images, operatorbundle.RelatedImage{Image: ri.Image, Digest: ri.Digest, Name: ri.Name})
	}
	return operatorbundle.Bundle{
		Package:         b.Package,
		ChannelName:     b.ChannelName,
		Version:         b.Version,
		VersionOriginal: b.VersionOriginal,
		CSVName:         b.CSVName,
		CSVDisplayName:  b.CSVDisplayName,
		CreationDate:    b.CreationDate,
		RelatedImages:   images,
	}
}

// jsonString returns a JSON-quoted, escaped string literal suitable for inlining a scalar
// value into a GraphQL query.
func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

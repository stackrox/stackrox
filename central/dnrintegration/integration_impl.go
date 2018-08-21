package dnrintegration

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// validateAndParseDirectorEndpoint parses the director endpoint into
// a URL object, making sure it's non-empty.
func validateAndParseDirectorEndpoint(directorEndpoint string) (*url.URL, error) {
	directorURL, err := url.Parse(directorEndpoint)
	if err != nil {
		return nil, fmt.Errorf("provided director endpoint '%s' not valid: %s",
			directorEndpoint, err)
	}

	// If they've provided a scheme other than https, don't allow it silently.
	if directorURL.Scheme != "" && directorURL.Scheme != "https" {
		return nil, fmt.Errorf("invalid URL scheme for D&R director: %s", directorURL.Scheme)
	}
	// Be kind if they haven't provided anything.
	directorURL.Scheme = "https"

	if directorURL.Host == "" {
		return nil, fmt.Errorf("invalid directorEndpoint '%s': empty host", directorEndpoint)
	}

	return directorURL, nil
}

func (d *dnrIntegrationImpl) makeAuthenticatedRequest(method, path string, params url.Values) ([]byte, error) {
	pathURL, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("path URL parsing: %s", err)
	}
	reqURL := d.directorURL.ResolveReference(pathURL)
	reqURL.RawQuery = params.Encode()

	req, err := http.NewRequest(method, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request to D&R: %s", err)
	}
	req.Header.Add("Authorization", d.authToken)
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to D&R failed: %s", err)
	}
	defer resp.Body.Close()

	// We read the results a little early so that, if the body exists,
	// we can print it out in the response for easier debuggability.
	results, readErr := ioutil.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errorBuilder := strings.Builder{}
		errorBuilder.WriteString(fmt.Sprintf("got error status code %d from D&R: %s",
			resp.StatusCode, resp.Status))
		if readErr != nil {
			errorBuilder.WriteString(fmt.Sprintf(" (body: %s)", results))
		}
		return nil, errors.New(errorBuilder.String())
	}
	if readErr != nil {
		return nil, fmt.Errorf("reading results: %s", err)
	}
	return results, nil
}

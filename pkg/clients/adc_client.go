package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"regexp"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

const adcScope = "https://management.azuredevcompute.io/.default"

var regionPattern = regexp.MustCompile(`^[a-z0-9]+$`)

type ResponseError struct {
	StatusCode int
	Body       string
}

func (e *ResponseError) Error() string {
	return fmt.Sprintf("request failed with status %d: %s", e.StatusCode, e.Body)
}

type ADCClient struct {
	endpoint   *url.URL
	credential azcore.TokenCredential
	httpClient *http.Client
}

func NewADCClient(credential azcore.TokenCredential, region string) (*ADCClient, error) {
	if credential == nil {
		return nil, errors.New("credential is required")
	}
	if !regionPattern.MatchString(region) {
		return nil, fmt.Errorf("invalid region %q: expected lowercase letters and digits, for example eastus2", region)
	}
	return &ADCClient{
		endpoint:   &url.URL{Scheme: "https", Host: fmt.Sprintf("management.%s.azuredevcompute.io", region)},
		credential: credential,
		httpClient: &http.Client{},
	}, nil
}

func (c *ADCClient) Do(ctx context.Context, method, path, apiVersion string, query url.Values, body, result any) (*http.Response, error) {
	u := c.endpoint.JoinPath(path)
	q := maps.Clone(query)
	if q == nil {
		q = url.Values{}
	}
	q.Set("api-version", apiVersion)
	u.RawQuery = q.Encode()

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), reqBody)
	if err != nil {
		return nil, err
	}

	token, err := c.credential.GetToken(ctx, policy.TokenRequestOptions{Scopes: []string{adcScope}})
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.Token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, &ResponseError{StatusCode: resp.StatusCode, Body: string(data)}
	}
	if result != nil && len(data) > 0 {
		if err := json.Unmarshal(data, result); err != nil {
			return nil, err
		}
	}
	return resp, nil
}

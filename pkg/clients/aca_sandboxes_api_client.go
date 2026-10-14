package clients

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Azure/vk-azure-container-apps-sandboxes/pkg/types"
)

const sandboxesAPIVersion = "2026-09-01-preview"

type AcaSandboxesAPIClientOptions struct {
	SubscriptionID string
	ResourceGroup  string
	SandboxGroup   string
	Location       string
}

type AcaSandboxesAPIClient struct {
	adc           *ADCClient
	sandboxesPath string
}

func NewAcaSandboxesAPIClient(adc *ADCClient, options AcaSandboxesAPIClientOptions) (*AcaSandboxesAPIClient, error) {
	if adc == nil {
		return nil, errors.New("ADC client is required")
	}
	if options.SubscriptionID == "" || options.ResourceGroup == "" || options.SandboxGroup == "" {
		return nil, errors.New("subscription ID, resource group and sandbox group are required")
	}
	sandboxesPath, err := url.JoinPath(
		"/subscriptions", options.SubscriptionID,
		"resourceGroups", options.ResourceGroup,
		"sandboxGroups", options.SandboxGroup,
		"sandboxes",
	)
	if err != nil {
		return nil, err
	}
	return &AcaSandboxesAPIClient{adc: adc, sandboxesPath: sandboxesPath}, nil
}

func (c *AcaSandboxesAPIClient) Create(ctx context.Context, sandbox *types.Sandbox) (*types.Sandbox, error) {
	if sandbox == nil {
		return nil, errors.New("sandbox is required")
	}
	var created types.Sandbox
	if _, err := c.adc.Do(ctx, http.MethodPost, c.sandboxesPath, sandboxesAPIVersion, nil, sandbox, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *AcaSandboxesAPIClient) Get(ctx context.Context, id string) (*types.Sandbox, error) {
	sandboxPath, err := c.sandboxPath(id)
	if err != nil {
		return nil, err
	}
	var sandbox types.Sandbox
	if _, err := c.adc.Do(ctx, http.MethodGet, sandboxPath, sandboxesAPIVersion, nil, nil, &sandbox); err != nil {
		return nil, err
	}
	return &sandbox, nil
}

func (c *AcaSandboxesAPIClient) List(ctx context.Context) ([]types.Sandbox, error) {
	selector := types.ManagedByLabelKey + "=" + types.ManagedByLabelValue
	path := c.sandboxesPath
	query := url.Values{"labels": {selector}}

	var sandboxes []types.Sandbox
	for {
		var page types.PagedSandbox
		if _, err := c.adc.Do(ctx, http.MethodGet, path, sandboxesAPIVersion, query, nil, &page); err != nil {
			return nil, err
		}
		sandboxes = append(sandboxes, page.Value...)
		if page.NextLink == "" {
			return sandboxes, nil
		}

		// Only follow the next link if it stays on our endpoint, so the bearer token is never sent elsewhere.
		next, err := url.Parse(page.NextLink)
		if err != nil {
			return nil, err
		}
		if next.Scheme != c.adc.endpoint.Scheme || next.Host != c.adc.endpoint.Host {
			return nil, fmt.Errorf("unexpected next link host in %q", page.NextLink)
		}
		path = next.EscapedPath()
		query = next.Query()
		query.Set("labels", selector)
	}
}

func (c *AcaSandboxesAPIClient) Delete(ctx context.Context, id string) error {
	sandboxPath, err := c.sandboxPath(id)
	if err != nil {
		return err
	}
	_, err = c.adc.Do(ctx, http.MethodDelete, sandboxPath, sandboxesAPIVersion, nil, nil, nil)
	return err
}

func (c *AcaSandboxesAPIClient) sandboxPath(id string) (string, error) {
	if id == "" {
		return "", errors.New("sandbox ID is required")
	}
	return url.JoinPath(c.sandboxesPath, url.PathEscape(id))
}

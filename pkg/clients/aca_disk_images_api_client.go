package clients

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Azure/vk-azure-container-apps-sandboxes/pkg/types"
)

const (
	diskImageReadyTimeout = 5 * time.Minute
	diskImagePollInterval = 3 * time.Second
)

type AcaDiskImagesAPIClient struct {
	adc            *ADCClient
	diskImagesPath string
}

func NewAcaDiskImagesAPIClient(adc *ADCClient, options AcaSandboxesAPIClientOptions) (*AcaDiskImagesAPIClient, error) {
	if adc == nil {
		return nil, errors.New("ADC client is required")
	}
	if options.SubscriptionID == "" || options.ResourceGroup == "" || options.SandboxGroup == "" {
		return nil, errors.New("subscription ID, resource group and sandbox group are required")
	}
	diskImagesPath, err := url.JoinPath(
		"/subscriptions", options.SubscriptionID,
		"resourceGroups", options.ResourceGroup,
		"sandboxGroups", options.SandboxGroup,
		"diskimages",
	)
	if err != nil {
		return nil, err
	}
	return &AcaDiskImagesAPIClient{adc: adc, diskImagesPath: diskImagesPath}, nil
}

func (c *AcaDiskImagesAPIClient) Create(ctx context.Context, diskImage *types.DiskImage) (*types.DiskImage, error) {
	if diskImage == nil {
		return nil, errors.New("disk image is required")
	}
	var created types.DiskImage
	if _, err := c.adc.Do(ctx, http.MethodPost, c.diskImagesPath, sandboxesAPIVersion, nil, diskImage, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

// If the ImagePullPolicy is "Always" or the Disk Image doesnt exist yet, this method will create it and wait until it is ready.
func (c *AcaDiskImagesAPIClient) CreateAndWait(ctx context.Context, diskImage *types.DiskImage) (*types.DiskImage, error) {
	created, err := c.Create(ctx, diskImage)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, diskImageReadyTimeout)
	defer cancel()

	for {
		current, err := c.Get(ctx, created.ID)
		if err != nil {
			return nil, fmt.Errorf("getting disk image %s: %w", created.ID, err)
		}
		if current.Status != nil {
			state := current.Status.State
			if strings.EqualFold(state, types.DiskImageStateReady) || strings.EqualFold(state, types.DiskImageStateSucceeded) {
				return current, nil
			}
			if strings.EqualFold(state, types.DiskImageStateFailed) {
				return nil, fmt.Errorf("disk image %s failed: %s", created.ID, current.Status.ErrorMessage)
			}
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("disk image %s was not ready within %s: %w", created.ID, diskImageReadyTimeout, ctx.Err())
		case <-time.After(diskImagePollInterval):
		}
	}
}

func (c *AcaDiskImagesAPIClient) Get(ctx context.Context, id string) (*types.DiskImage, error) {
	diskImagePath, err := c.diskImagePath(id)
	if err != nil {
		return nil, err
	}
	var diskImage types.DiskImage
	if _, err := c.adc.Do(ctx, http.MethodGet, diskImagePath, sandboxesAPIVersion, nil, nil, &diskImage); err != nil {
		return nil, err
	}
	return &diskImage, nil
}

func (c *AcaDiskImagesAPIClient) Delete(ctx context.Context, id string) error {
	diskImagePath, err := c.diskImagePath(id)
	if err != nil {
		return err
	}
	_, err = c.adc.Do(ctx, http.MethodDelete, diskImagePath, sandboxesAPIVersion, nil, nil, nil)
	return err
}

func (c *AcaDiskImagesAPIClient) diskImagePath(id string) (string, error) {
	if id == "" {
		return "", errors.New("disk image ID is required")
	}
	return url.JoinPath(c.diskImagesPath, url.PathEscape(id))
}

package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/mailru/easyjson"
	internalTypes "github.com/stackrox/rox/pkg/docker/types"
)

// ContainerList returns the list of containers in the docker host.
func (cli *Client) ContainerList(ctx context.Context, options types.ContainerListOptions) ([]*internalTypes.ContainerList, error) {
	query := url.Values{}

	if options.All {
		query.Set("all", "1")
	}

	if options.Limit != -1 {
		query.Set("limit", strconv.Itoa(options.Limit))
	}

	if options.Since != "" {
		query.Set("since", options.Since)
	}

	if options.Before != "" {
		query.Set("before", options.Before)
	}

	if options.Size {
		query.Set("size", "1")
	}

	if options.Filters.Len() > 0 {
		filterJSON, err := filters.ToParamWithVersion(cli.version, options.Filters)

		if err != nil {
			return nil, err
		}

		query.Set("filters", filterJSON)
	}

	resp, err := cli.get(ctx, "/containers/json", query, nil)
	if err != nil {
		return nil, err
	}

	var containers []*internalTypes.ContainerList
	err = json.NewDecoder(resp.body).Decode(&containers)
	ensureReaderClosed(resp)
	return containers, err
}

// ContainerInspect returns the container information and its raw representation.
func (cli *Client) ContainerInspect(ctx context.Context, containerID string, getSize bool) (*internalTypes.ContainerJSON, error) {
	query := url.Values{}
	if getSize {
		query.Set("size", "1")
	}
	serverResp, err := cli.get(ctx, "/containers/"+containerID+"/json", query, nil)
	if err != nil {
		if serverResp.statusCode == http.StatusNotFound {
			return nil, containerNotFoundError{containerID}
		}
		return nil, err
	}

	var container internalTypes.ContainerJSON
	if err := easyjson.UnmarshalFromReader(serverResp.body, &container); err != nil {
		return nil, err
	}
	ensureReaderClosed(serverResp)
	return &container, nil
}

package client

import (
	"encoding/json"
	"net/url"

	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

// Info returns information about the docker server.
func (cli *Client) Info(ctx context.Context) (types.Info, error) {
	var info types.Info
	serverResp, err := cli.get(ctx, "/info", url.Values{}, nil)
	if err != nil {
		return info, err
	}
	defer ensureReaderClosed(serverResp)

	if err := json.NewDecoder(serverResp.body).Decode(&info); err != nil {
		return info, errors.Wrapf(err, "error reading remote info")
	}

	return info, nil
}

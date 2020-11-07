package gatherer

import (
	"io/ioutil"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/networkgraph/defaultexternalsrcs"
)

func writeChecksumLocally(checksum []byte) error {
	if err := ioutil.WriteFile(defaultexternalsrcs.LocalChecksumFile, checksum, 0644); err != nil {
		return errors.Wrapf(err, "writing provider networks checksum %s", defaultexternalsrcs.LocalChecksumFile)
	}
	return nil
}

package backup

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/mathutil"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/central/db/transfer"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/download"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
)

const (
	idleTimeout = 5 * time.Minute
)

// Command defines the backup command.
func Command(full *bool) *cobra.Command {
	var output string
	c := &cobra.Command{
		Use:          "backup",
		SilenceUsage: true,
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			return Backup(flags.Timeout(c), output, *full)
		}),
	}

	c.Flags().StringVar(&output, "output", "", `where to write the backup.
If the provided path is a file path, the backup will be written to the file, overwriting it if it already exists. (The directory MUST exist.)
If the provided path is a directory, the backup will be saved in that directory with the server-provided filename.
If this argument is omitted, the backup will be saved in the current working directory with the server-provided filename.`)
	flags.AddTimeoutWithDefault(c, 1*time.Hour)
	return c
}

func parseUserProvidedOutput(userProvidedOutput string) (string, error) {
	if userProvidedOutput == "" {
		return "", nil
	}

	f, err := os.Stat(userProvidedOutput)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
		// If they specified a directory, it must exist.
		if strings.HasSuffix(userProvidedOutput, string(os.PathSeparator)) {
			return "", errors.Errorf("invalid output %q: directory does not exist", userProvidedOutput)
		}
		// Now we know they've provided a filename. We check to make sure the containing directory exists.
		containingDir := filepath.Dir(userProvidedOutput)
		dirExists, err := fileutils.Exists(containingDir)
		if err != nil {
			return "", err
		}
		if !dirExists {
			return "", errors.Errorf("invalid output %q: containing directory %q does not exist", userProvidedOutput, containingDir)
		}
		return userProvidedOutput, nil
	}
	if f.IsDir() {
		return stringutils.EnsureSuffix(userProvidedOutput, string(os.PathSeparator)), nil
	}
	return userProvidedOutput, nil
}

func getFilePath(respHeader http.Header, userProvidedOutput string) (string, error) {
	parsedOutputLocation, err := parseUserProvidedOutput(userProvidedOutput)
	if err != nil {
		return "", err
	}

	finalLocation := parsedOutputLocation
	// If they haven't specified a filename, fetch the filename from the server.
	if finalLocation == "" || strings.HasSuffix(finalLocation, string(os.PathSeparator)) {
		parsedFileName, err := download.ParseFilenameFromHeader(respHeader)
		if err != nil {
			return "", err
		}
		finalLocation = filepath.Join(finalLocation, parsedFileName)
	}
	return finalLocation, nil
}

// Backup creates central backup.
func Backup(timeout time.Duration, userProvidedOutput string, full bool) error {
	deadline := time.Now().Add(timeout)

	var endpoint string
	if full {
		endpoint = "/api/extensions/backup"
	} else {
		endpoint = "/db/backup"
	}

	req, err := common.NewHTTPRequestWithAuth(http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}

	client, err := common.GetHTTPClient(0)
	if err != nil {
		return err
	}

	reqCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req = req.WithContext(reqCtx)

	// Cancel the context if no headers have been received after `timeout`.
	t := time.AfterFunc(timeout, cancel)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if !t.Stop() {
		// The context will be canceled so we also can't do any reads.
		return errors.New("server took too long to send headers")
	}

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return errors.New("Invalid credentials. Please add/fix your credentials")
	default:
		return errors.Wrap(httputil.ResponseToError(resp), "Error when trying to get a backup.")
	}

	filename, err := getFilePath(resp.Header, userProvidedOutput)
	if err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return errors.Wrapf(err, "Could not create backup file %q", filename)
	}
	defer utils.IgnoreError(file.Close)

	if err := transfer.Copy(reqCtx, cancel, filename, mathutil.MaxInt64(0, resp.ContentLength), resp.Body, file, deadline, idleTimeout); err != nil {
		return err
	}

	fmt.Printf("Wrote backup file to %q\n", filename)
	return file.Close()
}

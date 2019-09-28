package backup

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/central/db/transfer"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/flags"
)

const (
	idleTimeout = 5 * time.Minute
)

// Command defines the db backup command
func Command() *cobra.Command {
	var output string
	c := &cobra.Command{
		Use:          "backup",
		Short:        "Save a snapshot of the DB as a backup.",
		Long:         "Save a snapshot of the DB as a backup.",
		SilenceUsage: true,
		RunE: func(c *cobra.Command, _ []string) error {
			return getBackup(flags.Timeout(c), output)
		},
	}

	c.Flags().StringVar(&output, "output", "", `where to write the backup to.
If the provided path is a file path, the backup will be written to the file, overwriting it if it exists already. (The directory MUST exist.)
If the provided path is a directory, the backup will be saved in that directory with the server-provided filename.
If this argument is omitted, the backup will be saved in the current working directory with the server-provided filename.`)
	return c
}

func parseFilenameFromHeader(header http.Header) (string, error) {
	data := header.Get("Content-Disposition")
	if data == "" {
		return data, fmt.Errorf("could not parse filename from header: %+v", header)
	}
	data = strings.TrimPrefix(data, "attachment; filename=")
	return strings.Trim(data, `"`), nil
}

func parseContentLengthFromHeader(header http.Header) (int64, error) {
	data := header.Get("Content-Length")
	if data == "" {
		return 0, nil
	}
	return strconv.ParseInt(data, 10, 64)
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
		parsedFileName, err := parseFilenameFromHeader(respHeader)
		if err != nil {
			return "", err
		}
		finalLocation = filepath.Join(finalLocation, parsedFileName)
	}
	return finalLocation, nil
}

func getBackup(timeout time.Duration, userProvidedOutput string) error {
	deadline := time.Now().Add(timeout)

	req, err := common.NewHTTPRequestWithAuth(http.MethodGet, "/db/backup", nil)
	if err != nil {
		return err
	}

	reqCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req = req.WithContext(reqCtx)

	client := common.GetHTTPClient(0)

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
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("Error with status code %d when trying to get a backup", resp.StatusCode)
		}
		return fmt.Errorf("Error with status code %d when trying to get a backup. Response body: %s", resp.StatusCode, string(body))
	}

	filename, err := getFilePath(resp.Header, userProvidedOutput)
	if err != nil {
		return err
	}

	totalSize, err := parseContentLengthFromHeader(resp.Header)
	if err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return errors.Wrapf(err, "Could not create backup file %q", filename)
	}
	defer utils.IgnoreError(file.Close)

	if err := transfer.Copy(reqCtx, cancel, filename, totalSize, resp.Body, file, deadline, idleTimeout); err != nil {
		return err
	}

	fmt.Printf("Wrote DB backup file to %q\n", filename)
	return file.Close()
}

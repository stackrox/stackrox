package restore

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/flags"
)

const (
	idleTimeout = 5 * time.Minute
)

// Command defines the db backup command
func Command() *cobra.Command {
	var file string
	c := &cobra.Command{
		Use:   "restore",
		Short: "Restore the Central DB from a local file.",
		Long:  "Restore the Central DB from a local file.",
		RunE: func(c *cobra.Command, _ []string) error {
			if file == "" {
				return fmt.Errorf("file to restore from must be specified")
			}
			return restore(file, flags.Timeout(c))
		},
	}

	c.Flags().StringVar(&file, "file", "", "file to restore the DB from")
	return c
}

func restore(filename string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	// Try to make the file path absolute, for better local file info, but don't insist on it.
	filePath, err := filepath.Abs(filename)
	if err != nil {
		filePath = filename
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer utils.IgnoreError(file.Close)

	err = ErrV2RestoreNotSupported
	if features.DBBackupRestoreV2.Enabled() {
		err = tryRestoreV2(file, deadline)
		if err == ErrV2RestoreNotSupported {
			fmt.Println("Your central instance does not support V2 database restore. Consider upgrading Central")
			fmt.Println("for a significantly improved backup/restore experience.")
		}
	}
	if err == ErrV2RestoreNotSupported {
		err = restoreV1(file, deadline)
	}

	if err != nil {
		return err
	}

	fmt.Println("Successfully restored DB")
	return nil
}

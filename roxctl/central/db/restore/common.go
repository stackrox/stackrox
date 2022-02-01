package restore

import (
	"fmt"
	"os"
	"time"

	"github.com/stackrox/rox/pkg/utils"
)

const (
	idleTimeout = 5 * time.Minute
)

func restore(filename string, timeout time.Duration, impl func(file *os.File, deadline time.Time) error) error {
	deadline := time.Now().Add(timeout)

	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer utils.IgnoreError(file.Close)

	if err := impl(file, deadline); err != nil {
		return err
	}

	fmt.Println("Successfully restored DB")
	return nil
}

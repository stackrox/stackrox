package common

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// ExecutePostgresCmd -- executes a command
func ExecutePostgresCmd(cmd *exec.Cmd) error {
	log.Info(cmd)

	cmd.Stderr = os.Stderr

	// Run the command
	err := cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()

	if exitError, ok := err.(*exec.ExitError); ok {
		log.Error(exitError)
		return err
	}

	return nil
}

// SetPostgresCmdEnv - sets command environment for postgres commands
func SetPostgresCmdEnv(cmd *exec.Cmd, sourceMap map[string]string, config *pgxpool.Config) {
	cmd.Env = os.Environ()

	if _, found := sourceMap["sslmode"]; found {
		cmd.Env = append(cmd.Env, fmt.Sprintf("PGSSLMODE=%s", sourceMap["sslmode"]))
	}
	if _, found := sourceMap["sslrootcert"]; found {
		cmd.Env = append(cmd.Env, fmt.Sprintf("PGSSLROOTCERT=%s", sourceMap["sslrootcert"]))
	}
	cmd.Env = append(cmd.Env, fmt.Sprintf("PGPASSWORD=%s", config.ConnConfig.Password))
}

package pgadmin

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/stackrox/rox/pkg/postgres"
)

// ExecutePostgresCmd -- executes a command
func ExecutePostgresCmd(cmd *exec.Cmd) error {
	log.Debug(cmd)

	cmd.Stderr = os.Stderr

	// Run the command
	err := cmd.Start()
	if err != nil {
		return err
	}

	err = cmd.Wait()
	if exitError, ok := err.(*exec.ExitError); ok {
		log.Errorf("Failure executing %q with %v", cmd, exitError)
		return exitError
	}

	log.Debug("Exiting execution of command")
	return nil
}

// SetPostgresCmdEnv - sets command environment for postgres commands
func SetPostgresCmdEnv(cmd *exec.Cmd, sourceMap map[string]string, config *postgres.Config) {
	cmd.Env = os.Environ()

	if _, found := sourceMap["sslmode"]; found {
		cmd.Env = append(cmd.Env, fmt.Sprintf("PGSSLMODE=%s", sourceMap["sslmode"]))
	}
	if _, found := sourceMap["sslrootcert"]; found {
		cmd.Env = append(cmd.Env, fmt.Sprintf("PGSSLROOTCERT=%s", sourceMap["sslrootcert"]))
	}
	cmd.Env = append(cmd.Env, fmt.Sprintf("PGPASSWORD=%s", config.ConnConfig.Password))
}

// GetConnectionOptions - returns a string slice with the common postgres connection options
func GetConnectionOptions(config *postgres.Config) []string {
	// Set the options for pg_dump from the connection config
	options := []string{
		"-U",
		config.ConnConfig.User,
		"-h",
		config.ConnConfig.Host,
		"-p",
		strconv.FormatUint(uint64(config.ConnConfig.Port), 10),
	}

	return options
}

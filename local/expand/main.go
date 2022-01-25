package main

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/tecbot/gorocksdb"
)

func main() {
	var (
		backup   string
		restored string
	)

	cmd := &cobra.Command{
		Use:   "rocksdb_restore",
		Short: "Restores rocksdb",
		RunE: func(*cobra.Command, []string) error {
			if backup == "" {
				return errors.New("backup flag needs to be specified")
			}
			if restored == "" {
				return errors.New("restore location needs to be specified")
			}

			fmt.Printf("Expanding database from %s to %s\n", backup, restored)
			be, err := gorocksdb.OpenBackupEngine(gorocksdb.NewDefaultOptions(), backup)
			if err != nil {
				panic(err)
			}
			if err := be.RestoreDBFromLatestBackup(restored, restored, gorocksdb.NewRestoreOptions()); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&backup, "backup", "", "location of the rocksdb backup")
	cmd.Flags().StringVar(&restored, "restored", "", "location to expand the new database")
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}

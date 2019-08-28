package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/sensor/upgrader/bundle"
	"github.com/stackrox/rox/sensor/upgrader/config"
	_ "github.com/stackrox/rox/sensor/upgrader/flags"
	"github.com/stackrox/rox/sensor/upgrader/runner"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
)

var (
	log = logging.LoggerForModule()

	validateBundlePath = flag.String("validate-bundle", "", "validate a bundle")
)

func validateBundle(ctx *upgradectx.UpgradeContext, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer utils.IgnoreError(f.Close)

	fi, err := f.Stat()
	if err != nil {
		return err
	}

	var contents bundle.Contents

	if fi.IsDir() {
		contents, err = bundle.ContentsFromDir(path)
	} else {
		contents, err = bundle.ContentsFromZIPData(f, fi.Size())
	}

	if err != nil {
		return errors.Wrapf(err, "reading bundle %s", path)
	}

	_, err = bundle.InstantiateBundle(ctx, contents)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	log.Infof("StackRox Sensor Upgrader, version %s", version.GetMainVersion())

	flag.Parse()

	upgraderCfg, err := config.Create()
	utils.Must(err)

	upgradeCtx, err := upgradectx.Create(upgraderCfg)
	utils.Must(err)

	if *validateBundlePath != "" {
		if err := validateBundle(upgradeCtx, *validateBundlePath); err != nil {
			fmt.Println("Bundle at", *validateBundlePath, "is invalid:")
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("Bundle at", *validateBundlePath, "is valid")
		os.Exit(0)
	}

	utils.Must(runner.Run(upgradeCtx))

	time.Sleep(15 * time.Second)
}

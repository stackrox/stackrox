package command

import (
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/tools/policyutil/common"
)

var (
	file             string
	dir              string
	out              string
	dryRun           bool
	readOnlySettings []string
)

// Command defines the upgrade command.
func Command() *cobra.Command {
	c := &cobra.Command{
		Use: fmt.Sprintf("upgrade [--%s <file> | --%s <dir>] [--%s <path>]",
			common.FileFlagName, common.DirectoryFlagName, common.OutputFlagName),
		Short: "Upgrade given policy(ies) to the current policy version",
		RunE:  upgrade,
	}

	c.PersistentFlags().StringVarP(
		&file,
		common.FileFlagName,
		common.FileFlagShorthand,
		"",
		"input file path")
	utils.Must(c.MarkPersistentFlagFilename(common.FileFlagName))

	c.PersistentFlags().StringVarP(
		&dir,
		common.DirectoryFlagName,
		common.DirectoryFlagShorthand,
		"",
		"input directory path")
	utils.Must(c.MarkPersistentFlagDirname(common.DirectoryFlagName))

	c.PersistentFlags().StringVarP(
		&out,
		common.OutputFlagName,
		common.OutputFlagShorthand,
		"",
		"output file or directory [required]")

	c.PersistentFlags().BoolVarP(
		&dryRun,
		common.DryRunFlagName,
		common.DryRunFlagShorthand,
		false,
		"dry run")

	c.PersistentFlags().StringArrayVarP(
		&readOnlySettings,
		common.ReadOnlyFlagName,
		"",
		[]string{common.None.String()},
		"one or more read-only policy settings to verify (mitre, criteria)")

	return c
}

func upgrade(_ *cobra.Command, _ []string) error {
	fileStat, fileErr := os.Stat(file)
	dirStat, dirErr := os.Stat(dir)

	// While most of the checks are either file or folder related, we perform as
	// many as we can here and group them together to avoid presenting one error
	// at a time to the user.
	var multiErr *multierror.Error
	if (file != "") == (dir != "") {
		multiErr = multierror.Append(multiErr, errors.Errorf("exactly one of '--%s' and '--%s' must be specified",
			common.FileFlagName, common.DirectoryFlagName))
	}
	if file != "" && fileErr != nil || fileErr == nil && fileStat.IsDir() {
		multiErr = multierror.Append(multiErr, errors.Errorf("'--%s' must be a valid file",
			common.FileFlagName))
	}
	if dir != "" && dirErr != nil || dirErr == nil && !dirStat.IsDir() {
		multiErr = multierror.Append(multiErr, errors.Errorf("'--%s' must be a valid directory",
			common.DirectoryFlagName))
	}
	if dirErr == nil && dirStat.IsDir() && out == "" {
		multiErr = multierror.Append(multiErr, errors.Errorf("'--%s' must be specified if '--%s' is set",
			common.OutputFlagName, common.DirectoryFlagName))
	}

	if len(readOnlySettings) == 0 {
		multiErr = multierror.Append(multiErr, errors.Errorf("'--%s' must be (mitre, criteria) or none",
			common.ReadOnlyFlagName))
	}

	if len(readOnlySettings) > 0 {
		for _, setting := range readOnlySettings {
			if _, ok := common.ReadOnlySettingStrToType[setting]; !ok {
				multiErr = multierror.Append(multiErr, errors.Errorf("'--%s' must be (mitre, criteria) or none",
					common.ReadOnlyFlagName))
			}
		}
	}

	if multiErr != nil {
		return multiErr
	}

	if dryRun {
		common.PrintLog("Running in dry-run mode")
	}

	// File is set, working with a single policy.
	if fileErr == nil && !fileStat.IsDir() {
		return upgradeSingle()
	}

	// Dir is set, working with a batch.
	if dirErr == nil && dirStat.IsDir() {
		return upgradeFolder()
	}

	return errors.Errorf("specify either '--%s' or '--%s'", common.FileFlagName, common.DirectoryFlagName)
}

func upgradeSingle() error {
	content, err := os.ReadFile(file)
	if err != nil {
		return errors.Wrap(err, "problem reading the file")
	}

	upgraded, err := upgradePolicyJSON(string(content))
	if err != nil {
		return errors.Wrap(err, "policy cannot be upgraded")
	}

	common.PrintVerboseLog("Policy successfully upgraded")
	common.PrintVerboseLog(common.DiffWrapped(string(content), upgraded))

	if out == "" {
		common.PrintVerboseLog("Upgraded policy is printed to stdout")
		common.PrintResult(upgraded)
		return nil
	}

	// Figure out the target file name if folder is passed as output.
	outfileStat, outfileErr := os.Stat(out)
	if outfileErr == nil && outfileStat.IsDir() {
		fileStat, _ := os.Stat(file)
		out = path.Join(out, fileStat.Name())
	}

	// Write upgraded policy to file.
	outfileStat, outfileErr = os.Stat(out)
	if outfileErr != nil && !os.IsNotExist(outfileErr) {
		return errors.New("problem writing the file")
	}

	if outfileErr == nil && !outfileStat.IsDir() {
		// Output file exists. Check with the user if they want to overwrite
		// it if running in the interactive mode.
		if common.Interactive {
			answer, err := common.ReadUserInput(fmt.Sprintf("File %q exists; overwrite? yes / No: ", out))
			if err != nil {
				return errors.Wrap(err, "cannot read user answer")
			}

			// Start with the "no" case because empty string
			// is a prefix of any string.
			switch {
			case strings.HasPrefix("no", strings.ToLower(answer)):
				common.PrintLog("Abort writing file")
				return nil
			case strings.HasPrefix("yes", strings.ToLower(answer)):
				// proceed with writing
			default:
				return errors.Errorf("%q is not a valid choice; please answer yes or no", answer)
			}
		} else {
			common.PrintLog("File %q exists; overwriting", out)
		}
	}

	if dryRun {
		common.PrintLog("Saving upgraded policy to %q SKIPPED (dry run)", out)
	} else {
		common.PrintVerboseLog("Saving upgraded policy to %q", out)
		err = os.WriteFile(out, []byte(upgraded+"\n"), 0644)
		if err != nil {
			return errors.Wrap(err, "writing file failed")
		}
		common.PrintVerboseLog("Save successful")
	}

	return nil
}

func upgradeFolder() error {
	outdirStat, outdirErr := os.Stat(out)
	if outdirErr == nil && !outdirStat.IsDir() {
		return errors.Errorf("cannot use %q because it is not a directory", out)
	}

	// Create output directory if necessary.
	if outdirErr != nil && os.IsNotExist(outdirErr) {
		err := os.MkdirAll(out, 0755)
		if err != nil {
			return errors.Wrap(err, "create output directory")
		}
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return errors.Wrap(err, "iterate input directory")
	}

	totalSucceeded := 0
	totalFailed := 0
	totalFailedUpgrades := 0
	totalOverwritten := 0
	totalIntact := 0
	totalUpgraded := 0

	totalFiles := 0
	for _, f := range files {
		if !f.IsDir() {
			totalFiles++
		}
	}

	var multiErr *multierror.Error
	for idx, file := range files {
		if file.IsDir() {
			continue
		}

		common.PrintVerboseLog("\n%v / %v: %q", idx+1, totalFiles, file.Name())

		source := path.Join(dir, file.Name())
		target := path.Join(out, file.Name())

		content, err := os.ReadFile(source)
		if err != nil {
			totalFailed++
			multiErr = multierror.Append(multiErr, errors.Wrapf(err, "problem reading file %q", source))
			continue
		}

		upgraded, err := upgradePolicyJSON(string(content))
		if err != nil {
			totalFailed++
			totalFailedUpgrades++
			multiErr = multierror.Append(multiErr, errors.Wrapf(err, "policy %q cannot be upgraded; not a policy?", file.Name()))
			continue
		}

		if strings.TrimSpace(upgraded) == strings.TrimSpace(string(content)) {
			totalIntact++
			common.PrintVerboseLog("No need to upgrade")
		} else {
			totalUpgraded++
			common.PrintVerboseLog("Policy successfully upgraded")
		}

		targetStat, targetErr := os.Stat(target)
		if targetErr != nil && !os.IsNotExist(targetErr) {
			totalFailed++
			multiErr = multierror.Append(multiErr, errors.Wrapf(targetErr, "problem writing file %q", target))
			continue
		}
		if targetErr == nil && !targetStat.IsDir() {
			// This might not be quite true if a write failure happens later
			// on, but then we have bigger problems.
			totalOverwritten++
			common.PrintVerboseLog("Overwriting file %q", target)
		}

		if dryRun {
			common.PrintVerboseLog("Saving policy to %q SKIPPED", target)
		} else {
			common.PrintVerboseLog("Saving upgraded policy to %q", target)
			err = os.WriteFile(target, []byte(upgraded+"\n"), 0644)
			if err != nil {
				totalFailed++
				multiErr = multierror.Append(multiErr, errors.Wrap(err, "writing file failed"))
				continue
			}
		}

		totalSucceeded++
	}

	common.PrintLog("\nProcessed %v file(s):\n"+
		"\t%v policy(-ies) upgraded\n"+
		"\t%v policy(-ies) do not require an upgrade\n"+
		"\t%v failed, of which %v due to upgrade error(s)",
		totalFiles, totalUpgraded, totalIntact, totalFailed, totalFailedUpgrades)

	if dryRun {
		common.PrintLog("No files written (dry run)")
		common.PrintLog("%v file(s) will be overwritten", totalOverwritten)
	} else {
		common.PrintLog("Generated %v file(s):\n"+
			"\t%v file(s) overwritten\n"+
			"\t%v file(s) new",
			totalSucceeded, totalOverwritten, totalSucceeded-totalOverwritten)
	}

	if multiErr != nil {
		common.PrintLog("\n%v", multiErr)
	}

	return nil
}

func upgradePolicyJSON(json string) (string, error) {
	var result string

	var policy storage.Policy
	err := jsonutil.JSONToProto(json, &policy)
	if err != nil {
		return result, errors.Wrap(err, "supplied text is not a valid policy")
	}

	err = policyversion.EnsureConvertedToLatest(&policy)
	if err != nil {
		return result, errors.Wrap(err, "upgrade error")
	}

	ensureReadOnlySettings(&policy)
	ensureMitreVectorSorted(&policy)

	result, err = jsonutil.ProtoToJSON(&policy, jsonutil.OptUnEscape)
	if err != nil {
		return result, errors.Wrap(err, "upgraded policy can't be serialized to JSON")
	}

	return result, nil
}

func ensureReadOnlySettings(policy *storage.Policy) {
	for _, setting := range readOnlySettings {
		if setting == common.Mitre.String() {
			policy.MitreVectorsLocked = true
		}

		if setting == common.Criteria.String() {
			policy.CriteriaLocked = true
		}
	}
}

func ensureMitreVectorSorted(policy *storage.Policy) {
	for _, vector := range policy.GetMitreAttackVectors() {
		techniques := vector.GetTechniques()
		sort.Slice(techniques, func(i, j int) bool {
			return techniques[i] < techniques[j]
		})
	}

	vectors := policy.GetMitreAttackVectors()
	sort.Slice(vectors, func(i, j int) bool {
		return vectors[i].GetTactic() < vectors[j].GetTactic()
	})
}

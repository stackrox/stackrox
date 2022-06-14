package docker

import (
	"encoding/json"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/generated/internalapi/compliance"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/compliance/checks/common"
	"github.com/stackrox/stackrox/pkg/compliance/checks/standards"
	"github.com/stackrox/stackrox/pkg/compliance/framework"
	"github.com/stackrox/stackrox/pkg/compliance/msgfmt"
	internalTypes "github.com/stackrox/stackrox/pkg/docker/types"
)

func init() {
	standards.RegisterChecksForStandard(standards.CISDocker, map[string]*standards.CheckAndMetadata{
		standards.CISDockerCheckName("2_1"): {
			CheckFunc: common.CheckWithDockerData(networkRestrictionCheck),
			Metadata: &standards.Metadata{
				InterpretationText: "StackRox checks that ICC is not enabled for the bridge network",
				TargetKind:         framework.NodeKind,
			},
		},
		standards.CISDockerCheckName("2_2"): genericDockerCommandlineCheck("log-level", "info", "info", common.Matches),
		standards.CISDockerCheckName("2_3"): genericDockerCommandlineCheck("iptables", "false", "true", common.NotMatches),
		standards.CISDockerCheckName("2_4"): {
			CheckFunc: common.CheckNoInsecureRegistries,
			Metadata: &standards.Metadata{
				InterpretationText: `StackRox checks that no insecure Docker Registries are configured on any host, except for those with an IP in a private subnet (such as 127.0.0.0/8 or 10.0.0.0/8)`,
				TargetKind:         framework.NodeKind,
			},
		},
		standards.CISDockerCheckName("2_5"): dockerInfoCheck(aufs),

		standards.CISDockerCheckName("2_6"): {
			CheckFunc: tlsVerifyCheck(),
		},
		standards.CISDockerCheckName("2_7"): genericDockerCommandlineCheck("default-ulimit", "", "", common.Info),
		standards.CISDockerCheckName("2_8"): dockerInfoCheck(userNamespaceInfo),

		standards.CISDockerCheckName("2_9"):  genericDockerCommandlineCheck("cgroup-parent", "", "", common.Info),
		standards.CISDockerCheckName("2_10"): genericDockerCommandlineCheck("storage-opt", "dm.basesize", "", common.NotContains),
		standards.CISDockerCheckName("2_11"): genericDockerCommandlineCheck("authorization-plugin", "", "", common.Set),
		standards.CISDockerCheckName("2_12"): dockerInfoCheck(remoteLogging),
		standards.CISDockerCheckName("2_13"): dockerInfoCheck(liveRestoreEnabled),
		standards.CISDockerCheckName("2_14"): genericDockerCommandlineCheck("userland-proxy", "false", "true", common.Matches),
		standards.CISDockerCheckName("2_15"): dockerInfoCheck(daemonSeccomp),
		standards.CISDockerCheckName("2_16"): dockerInfoCheck(disableExperimental),
		standards.CISDockerCheckName("2_17"): genericDockerCommandlineCheck("no-new-privileges", "false", "false", common.NotMatches),
	})
}

func networkRestrictionCheck(data *internalTypes.Data) []*storage.ComplianceResultValue_Evidence {
	if data.BridgeNetwork.Options["com.docker.network.bridge.enable_icc"] == "true" {
		return common.FailListf("Enable icc is true on bridge network")
	}

	return common.PassListf("Enable icc is false on bridge network")
}

func dockerInfoCheck(f func(info types.Info) []*storage.ComplianceResultValue_Evidence, optInterpretation ...string) *standards.CheckAndMetadata {
	var interpretationText string
	if len(optInterpretation) > 0 {
		interpretationText = optInterpretation[0]
	}
	return &standards.CheckAndMetadata{
		CheckFunc: common.CheckWithDockerData(
			func(data *internalTypes.Data) []*storage.ComplianceResultValue_Evidence {
				return f(data.Info)
			}),
		Metadata: &standards.Metadata{
			InterpretationText: interpretationText,
			TargetKind:         framework.NodeKind,
		},
	}
}

func aufs(info types.Info) []*storage.ComplianceResultValue_Evidence {
	if strings.Contains(info.Driver, "aufs") {
		return common.FailList("'aufs' is currently configured as the storage driver")
	}
	return common.PassListf("Storage driver is set to %q", info.Driver)
}

func daemonSeccomp(info types.Info) []*storage.ComplianceResultValue_Evidence {
	for _, opt := range info.SecurityOptions {
		if strings.HasPrefix("seccomp", opt) {
			return common.NoteListf("Seccomp profile is set to %q", opt)
		}
	}
	return common.NoteList("Seccomp profile is set to Docker Default")
}

func disableExperimental(info types.Info) []*storage.ComplianceResultValue_Evidence {
	if info.ExperimentalBuild {
		return common.FailListf("Docker is running in experimental mode")
	}
	return common.PassList("Docker is not running in experimental mode")
}

func liveRestoreEnabled(info types.Info) []*storage.ComplianceResultValue_Evidence {
	if !info.LiveRestoreEnabled {
		return common.FailListf("Live restore is not enabled")
	}
	return common.PassList("Live restore is enabled")
}

func remoteLogging(info types.Info) []*storage.ComplianceResultValue_Evidence {
	if info.LoggingDriver == "json-file" {
		return common.FailListf("Logging driver 'json-file' does not allow for remote logging")
	}
	return common.PassListf("Logging driver is set to %q", info.LoggingDriver)
}

func userNamespaceInfo(info types.Info) []*storage.ComplianceResultValue_Evidence {
	for _, opt := range info.SecurityOptions {
		if opt == "userns" {
			return common.PassListf("userns is set")
		}
	}
	return common.FailListf("userns is not set in security options: %s", msgfmt.FormatStrings(info.SecurityOptions...))
}

func getDockerdProcess(complianceData *standards.ComplianceData) (*compliance.CommandLine, map[string]interface{}, error) {
	var dockerdProcess *compliance.CommandLine
	for _, c := range complianceData.CommandLines {
		if strings.Contains(c.Process, "dockerd") {
			dockerdProcess = c
		}
	}
	if dockerdProcess == nil {
		return nil, nil, errors.New("Could not find a process that matched 'dockerd'")
	}
	// Get Daemon if it exists
	var daemonConfigFile *compliance.File
	for _, a := range dockerdProcess.Args {
		if a.Key == "config-file" {
			daemonConfigFile = a.File
		}
	}

	config := make(map[string]interface{})
	if daemonConfigFile != nil {
		if err := json.Unmarshal(daemonConfigFile.Content, &config); err != nil {
			return nil, nil, errors.Wrapf(err, "Unable to unmarshal Daemon config file %q", daemonConfigFile.Path)
		}
	}
	return dockerdProcess, config, nil
}

// Handle the command line inputs as well as if the daemon exists
// Here are all the handlers for just the daemon portion
func genericDockerCommandlineCheck(key, target, defaultVal string, evalFunc common.CommandEvaluationFunc) *standards.CheckAndMetadata {
	return &standards.CheckAndMetadata{
		CheckFunc: func(complianceData *standards.ComplianceData) []*storage.ComplianceResultValue_Evidence {
			dockerdProcess, config, err := getDockerdProcess(complianceData)
			if err != nil {
				return common.FailList(err.Error())
			}
			values := common.GetValuesForCommandFromFlagsAndConfig(dockerdProcess.Args, config, key)
			return evalFunc(values, key, target, defaultVal)
		},
	}
}

func tlsVerifyCheck() standards.Check {
	return func(complianceData *standards.ComplianceData) []*storage.ComplianceResultValue_Evidence {
		dockerdProcess, config, err := getDockerdProcess(complianceData)
		if err != nil {
			return common.FailList(err.Error())
		}
		args := dockerdProcess.Args
		values := common.GetValuesForCommandFromFlagsAndConfig(args, config, "host")

		var exposedOverTCP bool
		for _, v := range values {
			if strings.HasPrefix(v, "tcp://") {
				exposedOverTCP = true
			}
		}
		if !exposedOverTCP {
			return common.PassList("Docker daemon is not exposed over TCP")
		}

		var evidence []string
		tlsVerify := common.GetValuesForCommandFromFlagsAndConfig(args, config, "tlsverify")
		if len(tlsVerify) == 0 {
			evidence = append(evidence, "tlsverify is not set")
		}
		tlscacert := common.GetValuesForCommandFromFlagsAndConfig(args, config, "tlscacert")
		if len(tlscacert) == 0 {
			evidence = append(evidence, "tlscacert is not set")
		}
		tlscert := common.GetValuesForCommandFromFlagsAndConfig(args, config, "tlscert")
		if len(tlscert) == 0 {
			evidence = append(evidence, "tlscert is not set")
		}
		tlskey := common.GetValuesForCommandFromFlagsAndConfig(args, config, "tlskey")
		if len(tlskey) == 0 {
			evidence = append(evidence, "tlskey is not set")
		}
		if len(evidence) == 0 {
			return common.PassListf("TLS is properly set for the Docker Daemon (TLSCaCert=%s, TLSCert=%s, TLSKey=%s)", tlscacert, tlscert, tlskey)
		}

		var results []*storage.ComplianceResultValue_Evidence
		for _, e := range evidence {
			results = append(results, common.Fail(e))
		}
		return results
	}
}

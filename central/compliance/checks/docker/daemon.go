package docker

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/checks/msgfmt"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/docker"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

func init() {
	framework.MustRegisterChecks(
		networkRestrictionCheck(),
		genericDockerCommandlineCheck("CIS_Docker_v1_1_0:2_2", "log-level", "info", "info", common.Matches),
		genericDockerCommandlineCheck("CIS_Docker_v1_1_0:2_3", "iptables", "false", "true", common.Matches),
		framework.NewCheckFromFunc(
			framework.CheckMetadata{
				ID:                 "CIS_Docker_v1_1_0:2_4",
				Scope:              framework.NodeKind,
				DataDependencies:   []string{"DockerData"},
				InterpretationText: `StackRox checks that no insecure Docker Registries are configured on any host, except for those with an IP in a private subnet (such as 127.0.0.0/8 or 10.0.0.0/8)`,
			},
			common.CheckNoInsecureRegistries),
		dockerInfoCheck("CIS_Docker_v1_1_0:2_5", aufs),
		tlsVerifyCheck("CIS_Docker_v1_1_0:2_6"),
		genericDockerCommandlineCheck("CIS_Docker_v1_1_0:2_7", "default-ulimit", "", "", common.Set),
		dockerInfoCheck("CIS_Docker_v1_1_0:2_8", userNamespaceInfo),
		genericDockerCommandlineCheck("CIS_Docker_v1_1_0:2_9", "cgroup-parent", "", "", common.Matches),
		genericDockerCommandlineCheck("CIS_Docker_v1_1_0:2_10", "storage-opt", "dm.basesize", "", common.Contains),
		genericDockerCommandlineCheck("CIS_Docker_v1_1_0:2_11", "authorization-plugin", "", "", common.Set),
		dockerInfoCheck("CIS_Docker_v1_1_0:2_12", remoteLogging),
		genericDockerCommandlineCheck("CIS_Docker_v1_1_0:2_13", "disable-legacy-registry", "", "", common.Set),
		dockerInfoCheck("CIS_Docker_v1_1_0:2_14", liveRestoreEnabled),
		genericDockerCommandlineCheck("CIS_Docker_v1_1_0:2_15", "userland-proxy", "false", "true", common.Matches),
		dockerInfoCheck("CIS_Docker_v1_1_0:2_16", daemonSeccomp),
		dockerInfoCheck("CIS_Docker_v1_1_0:2_17", disableExperimental),
		genericDockerCommandlineCheck("CIS_Docker_v1_1_0:2_18", "no-new-privileges", "", "", common.Set),
	)
}

func networkRestrictionCheck() framework.Check {
	md := framework.CheckMetadata{
		ID:                 "CIS_Docker_v1_1_0:2_1",
		Scope:              framework.NodeKind,
		InterpretationText: "StackRox checks that ICC is not enabled for the bridge network",
	}
	return framework.NewCheckFromFunc(md, common.PerNodeCheckWithDockerData(
		func(ctx framework.ComplianceContext, data *docker.Data) {
			if data.BridgeNetwork.Options["com.docker.network.bridge.enable_icc"] == "true" {
				framework.Failf(ctx, "Enable icc is true on bridge network")
			} else {
				framework.Passf(ctx, "Enable icc is false on bridge network")
			}
		}))
}

func dockerInfoCheck(name string, f func(ctx framework.ComplianceContext, info types.Info), optInterpretation ...string) framework.Check {
	var interpretationText string
	if len(optInterpretation) > 0 {
		interpretationText = optInterpretation[0]
	}
	return framework.NewCheckFromFunc(
		framework.CheckMetadata{ID: name, Scope: framework.NodeKind, InterpretationText: interpretationText},
		common.PerNodeCheckWithDockerData(
			func(ctx framework.ComplianceContext, data *docker.Data) {
				f(ctx, data.Info)
			}))
}

func aufs(ctx framework.ComplianceContext, info types.Info) {
	if strings.Contains(info.Driver, "aufs") {
		framework.FailNow(ctx, "aufs is currently configured as the storage driver")
	}
	framework.Passf(ctx, "Storage driver is set to %q", info.Driver)
}

func daemonSeccomp(ctx framework.ComplianceContext, info types.Info) {
	for _, opt := range info.SecurityOptions {
		if strings.HasPrefix("seccomp", opt) {
			framework.NoteNowf(ctx, "Seccomp profile is set to %q", opt)
		}
	}
	framework.Note(ctx, "Seccomp profile is set to Docker Default")
}

func disableExperimental(ctx framework.ComplianceContext, info types.Info) {
	if info.ExperimentalBuild {
		framework.FailNowf(ctx, "Docker is running in experimental mode")
	}
	framework.Pass(ctx, "Docker is not running in experimental mode")
}

func liveRestoreEnabled(ctx framework.ComplianceContext, info types.Info) {
	if !info.LiveRestoreEnabled {
		framework.FailNowf(ctx, "Live restore is not enabled")
	}
	framework.Pass(ctx, "Live restore is enabled")
}

func remoteLogging(ctx framework.ComplianceContext, info types.Info) {
	if info.LoggingDriver == "json-file" {
		framework.FailNowf(ctx, "Logging driver 'json-file' does not allow for remote logging")
	}
	framework.Passf(ctx, "Logging driver is set to %q", info.LoggingDriver)
}

func userNamespaceInfo(ctx framework.ComplianceContext, info types.Info) {
	for _, opt := range info.SecurityOptions {
		if opt == "userns" {
			framework.PassNowf(ctx, "userns is set")
		}
	}
	framework.Failf(ctx, "userns is not set in security options: %s", msgfmt.FormatStrings(info.SecurityOptions...))
}

func getDockerdProcess(ret *compliance.ComplianceReturn) (*compliance.CommandLine, map[string]interface{}, error) {
	var dockerdProcess *compliance.CommandLine
	for _, c := range ret.CommandLines {
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
			return nil, nil, fmt.Errorf("Unable to unmarshal Daemon config file %q: %v", daemonConfigFile.Path, err)
		}
	}
	return dockerdProcess, config, nil
}

// Handle the command line inputs as well as if the daemon exists
// Here are all the handlers for just the daemon portion
func genericDockerCommandlineCheck(name string, key, target, defaultVal string, evalFunc common.CommandEvaluationFunc) framework.Check {
	return framework.NewCheckFromFunc(framework.CheckMetadata{ID: name, Scope: framework.NodeKind}, common.PerNodeCheck(
		func(ctx framework.ComplianceContext, ret *compliance.ComplianceReturn) {
			dockerdProcess, config, err := getDockerdProcess(ret)
			if err != nil {
				framework.FailNow(ctx, err.Error())
			}
			values := common.GetValuesForCommandFromFlagsAndConfig(dockerdProcess.Args, config, key)
			evalFunc(ctx, values, key, target, defaultVal)
		}))
}

func tlsVerifyCheck(name string) framework.Check {
	return framework.NewCheckFromFunc(framework.CheckMetadata{ID: name, Scope: framework.NodeKind}, common.PerNodeCheck(
		func(ctx framework.ComplianceContext, ret *compliance.ComplianceReturn) {
			dockerdProcess, config, err := getDockerdProcess(ret)
			if err != nil {
				framework.FailNow(ctx, err.Error())
			}
			args := dockerdProcess.Args
			values := common.GetValuesForCommandFromFlagsAndConfig(args, config, "host")
			for _, v := range values {
				if strings.Contains(v, "fd://") {
					framework.PassNowf(ctx, "Docker daemon is exposed over %q and not over TCP", v)
				}
			}
			if len(values) > 0 {
				framework.PassNowf(ctx, "host is set to %s", msgfmt.FormatStrings(values...))
			}

			var evidence []string
			tlsVerify := common.GetValuesForCommandFromFlagsAndConfig(args, config, "tlsverify")
			if len(tlsVerify) == 0 {
				evidence = append(evidence, "tlsverify is not set")
			}
			tlscacert := common.GetValuesForCommandFromFlagsAndConfig(args, config, "tlscacert")
			if len(tlsVerify) == 0 {
				evidence = append(evidence, "tlscacert is not set")
			}
			tlscert := common.GetValuesForCommandFromFlagsAndConfig(args, config, "tlscert")
			if len(tlsVerify) == 0 {
				evidence = append(evidence, "tlscert is not set")
			}
			tlskey := common.GetValuesForCommandFromFlagsAndConfig(args, config, "tlskey")
			if len(tlsVerify) == 0 {
				evidence = append(evidence, "tlskey is not set")
			}
			if len(evidence) == 0 {
				framework.PassNowf(ctx, "TLS is properly set for the Docker Daemon (TLSCaCert=%s, TLSCert=%s, TLSKey=%s)", tlscacert, tlscert, tlskey)
			} else {
				for _, e := range evidence {
					framework.Fail(ctx, e)
				}
			}
		}))
}

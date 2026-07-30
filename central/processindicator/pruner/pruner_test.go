package pruner

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stackrox/rox/central/processindicator"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

const (
	rabbitMQPodID       = "rabbitmq-app-7c47649749-twr9c"
	rabbitMQContainerID = "279dc850c0a9"
)

var (
	deterministicRabbitMQProcess = &storage.ProcessIndicator{
		Id:    uuid.NewV4().String(),
		PodId: rabbitMQPodID,
		Signal: &storage.ProcessSignal{
			ContainerId: rabbitMQContainerID,
			Name:        "beam.smp",
			Args: "-W w -A 64 -MBas ageffcbf -MHas ageffcbf -MBlmbcs 512 -MHlmbcs 512 -MMmcs 30 -P 1048576 " +
				"-t 5000000 -stbt db -zdbbl 128000 -K true -B i -- -root /usr/lib/erlang -progname erl -- " +
				"-home /var/lib/rabbitmq -- -pa /usr/lib/rabbitmq/lib/rabbitmq_server-3.7.8/ebin -noshell -noinput -s " +
				"rabbit boot -sname rabbit@rabbitmq-app-7c47649749-twr9c -boot start_sasl -conf /etc/rabbitmq/rabbitmq.conf " +
				"-conf_dir /var/lib/rabbitmq/config -conf_script_dir /usr/lib/rabbitmq/bin -conf_schema_dir /var/lib/rabbitmq/schema -conf_advanced " +
				"/etc/rabbitmq/advanced.config -kernel inet_default_connect_options [{nodelay,true}] -sasl errlog_type error -sasl sasl_error_logger " +
				"tty -rabbit lager_log_root \"/var/log/rabbitmq\" -rabbit lager_default_file tty -rabbit lager_upgrade_file " +
				"tty -rabbit enabled_plugins_file \"/etc/rabbitmq/enabled_plugins\" -rabbit plugins_dir " +
				"\"/usr/lib/rabbitmq/plugins:/usr/lib/rabbitmq/lib/rabbitmq_server-3.7.8/plugins\" -rabbit " +
				"plugins_expand_dir \"/var/lib/rabbitmq/mnesia/rabbit@rabbitmq-app-7c47649749-twr9c-plugins-expand\" " +
				"-os_mon start_cpu_sup false -os_mon start_disksup false -os_mon start_memsup false -mnesia dir " +
				"\"/var/lib/rabbitmq/mnesia/rabbit@rabbitmq-app-7c47649749-twr9c\" -kernel inet_dist_listen_min 25672 " +
				"-kernel inet_dist_listen_max 25672",
			ExecFilePath: "/usr/lib/erlang/erts-9.3.3.3/bin/beam.smp",
		},
	}
)

func rabbitMQBeamSMPProcess() *storage.ProcessIndicator {
	return &storage.ProcessIndicator{
		Id:    uuid.NewV4().String(),
		PodId: rabbitMQPodID,
		Signal: &storage.ProcessSignal{
			ContainerId:  rabbitMQContainerID,
			Name:         "beam.smp",
			Args:         fmt.Sprintf("-- -root /usr/lib/erlang -progname erl -- -home /var/lib/rabbitmq -- -sname epmd-starter-%d -noshell -eval halt()", rand.Intn(int(math.Pow10(9)))),
			ExecFilePath: "/usr/lib/erlang/erts-9.3.3.3/bin/beam.smp",
		},
	}
}

func processToIDAndArgs(process *storage.ProcessIndicator) processindicator.IDAndArgs {
	return processindicator.IDAndArgs{
		ID:   process.GetId(),
		Args: process.GetSignal().GetArgs(),
	}
}

func TestRabbitMQPruning(t *testing.T) {
	var processes []processindicator.IDAndArgs
	processes = append(processes, processToIDAndArgs(deterministicRabbitMQProcess))
	for range 1000 {
		processes = append(processes, processToIDAndArgs(rabbitMQBeamSMPProcess()))
	}
	pruner := NewFactory(1, time.Second).StartPruning()
	prunedIDs := pruner.Prune(processes)
	pruner.Finish()
	assert.Len(t, prunedIDs, len(processes)-2)
	assert.NotContains(t, prunedIDs, deterministicRabbitMQProcess.GetId())
}

// jaccardSimilarity computes Jaccard similarity between two word sets for
// test assertions. This reproduces the old pruner's algorithm to demonstrate
// cases where it incorrectly merges security-relevant commands.
func jaccardSimilarity(a, b map[string]struct{}) float64 {
	if len(a) > len(b) {
		a, b = b, a
	}
	var intersection int
	for w := range a {
		if _, ok := b[w]; ok {
			intersection++
		}
	}
	union := len(a) + len(b) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func toWordSet(args string) map[string]struct{} {
	set := make(map[string]struct{})
	for _, w := range strings.Fields(args) {
		set[numericRegex.ReplaceAllString(w, "#")] = struct{}{}
	}
	return set
}

// TestJaccardWouldMergeSecurityRelevantCommands demonstrates that the old
// Jaccard similarity approach (threshold 0.6) merges commands that differ in
// a single security-critical word. "kubectl get pods" and "kubectl get secrets"
// share enough flag words to exceed the 0.6 threshold, so the old pruner
// would delete the "secrets" indicator — destroying evidence that someone
// enumerated secrets in the cluster.
func TestJaccardWouldMergeSecurityRelevantCommands(t *testing.T) {
	getPods := "kubectl get pods --namespace kube-system -o json --field-selector status.phase=Running"
	getSecrets := "kubectl get secrets --namespace kube-system -o json --field-selector status.phase=Running"
	getNodes := "kubectl get nodes --namespace kube-system -o json --field-selector status.phase=Running"

	// Prove that the old Jaccard algorithm would consider these "similar"
	// (>= 0.6 threshold) and merge them.
	sim := jaccardSimilarity(toWordSet(getPods), toWordSet(getSecrets))
	assert.Greater(t, sim, 0.6,
		"expected Jaccard similarity between 'get pods' and 'get secrets' to exceed the old 0.6 threshold (got %f); "+
			"this proves the old pruner would have merged them", sim)

	sim = jaccardSimilarity(toWordSet(getPods), toWordSet(getNodes))
	assert.Greater(t, sim, 0.6,
		"expected Jaccard similarity between 'get pods' and 'get nodes' to exceed 0.6 (got %f)", sim)

	// Now verify that the current normalized-exact-match pruner preserves
	// all three as distinct, because "pods", "secrets", and "nodes" are
	// different words.
	processes := []processindicator.IDAndArgs{
		{ID: "get-pods", Args: getPods},
		{ID: "get-secrets", Args: getSecrets},
		{ID: "get-nodes", Args: getNodes},
	}
	pruner := NewFactory(1, time.Second).StartPruning()
	prunedIDs := pruner.Prune(processes)
	pruner.Finish()

	assert.Empty(t, prunedIDs, "all three semantically distinct kubectl commands should survive pruning")
}

func TestSemanticallyDistinctCommandsPreserved(t *testing.T) {
	processes := []processindicator.IDAndArgs{
		{ID: "1", Args: "kubectl get pods --namespace kube-system -o wide --timeout 30"},
		{ID: "2", Args: "kubectl get deployments --namespace kube-system -o wide --timeout 30"},
		{ID: "3", Args: "kubectl get secrets --namespace kube-system -o wide --timeout 30"},
		{ID: "4", Args: "kubectl get configmaps --namespace kube-system -o wide --timeout 30"},
		{ID: "5", Args: "kubectl get pods --namespace kube-system -o wide --timeout 45"},
		{ID: "6", Args: "kubectl get pods --namespace default -o wide --timeout 30"},
		{ID: "7", Args: "kubectl get secrets --namespace default -o wide --timeout 30"},
	}
	pruner := NewFactory(1, time.Second).StartPruning()
	prunedIDs := pruner.Prune(processes)
	pruner.Finish()

	kept := make(map[string]bool)
	prunedSet := make(map[string]bool)
	for _, id := range prunedIDs {
		prunedSet[id] = true
	}
	for _, p := range processes {
		if !prunedSet[p.ID] {
			kept[p.ID] = true
		}
	}

	// "pods" and "deployments" and "secrets" and "configmaps" are different resources.
	// With normalized-exact-match, the only merges are digit-only differences:
	// ID 1 (pods, kube-system, timeout 30) and ID 5 (pods, kube-system, timeout 45)
	// normalize to the same string since 30→# and 45→#.
	// ID 3 (secrets, kube-system) and ID 7 (secrets, default) differ in namespace,
	// so they stay distinct.
	// ID 6 (pods, default) differs from ID 1 (pods, kube-system) only in namespace,
	// so they stay distinct.
	assert.True(t, kept["1"], "first 'kubectl get pods' should be kept")
	assert.True(t, kept["2"], "'kubectl get deployments' should be kept (different resource)")
	assert.True(t, kept["3"], "'kubectl get secrets kube-system' should be kept")
	assert.True(t, kept["4"], "'kubectl get configmaps' should be kept")
	assert.True(t, prunedSet["5"], "second 'kubectl get pods' with different timeout should be pruned (digit-only diff)")
	assert.True(t, kept["6"], "'kubectl get pods default' should be kept (different namespace)")
	assert.True(t, kept["7"], "'kubectl get secrets default' should be kept (different namespace)")
}

func BenchmarkRabbitMQPruning(b *testing.B) {
	var processes []processindicator.IDAndArgs
	processes = append(processes, processToIDAndArgs(deterministicRabbitMQProcess))
	for range 1000000 {
		processes = append(processes, processToIDAndArgs(rabbitMQBeamSMPProcess()))
	}
	for b.Loop() {
		pruner := NewFactory(1, time.Second).StartPruning()
		pruner.Prune(processes)
		pruner.Finish()
	}
}

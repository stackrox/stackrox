package pruner

import (
	"fmt"
	"math"
	"math/rand"
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
	rand.Seed(time.Now().UnixNano())
	var processes []processindicator.IDAndArgs
	processes = append(processes, processToIDAndArgs(deterministicRabbitMQProcess))
	for i := 0; i < 1000; i++ {
		processes = append(processes, processToIDAndArgs(rabbitMQBeamSMPProcess()))
	}
	pruner := NewFactory(1, time.Second).StartPruning()
	prunedIDs := pruner.Prune(processes)
	pruner.Finish()
	assert.Len(t, prunedIDs, len(processes)-2)
	assert.NotContains(t, prunedIDs, deterministicRabbitMQProcess.GetId())
}

func BenchmarkRabbitMQPruning(b *testing.B) {
	var processes []processindicator.IDAndArgs
	processes = append(processes, processToIDAndArgs(deterministicRabbitMQProcess))
	for i := 0; i < 1000000; i++ {
		processes = append(processes, processToIDAndArgs(rabbitMQBeamSMPProcess()))
	}
	for i := 0; i < b.N; i++ {
		pruner := NewFactory(1, time.Second).StartPruning()
		pruner.Prune(processes)
		pruner.Finish()
	}
}

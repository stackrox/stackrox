
package {{.packageName}}

import (
	"github.com/stackrox/rox/central/backgroundmigrations/migrations"
	"github.com/stackrox/rox/central/backgroundmigrations/types"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

func init() {
	migrations.MustRegister(types.BackgroundMigration{
		StartingSeqNum:     {{.startSequenceNumber}},
		VersionAfterSeqNum: {{.nextSeqNum}},
		Description:        {{printf "%q" .description}},
		Run:                run,
	})
}

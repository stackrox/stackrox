package runner

import (
	// Import these packages to trigger the registration.
	_ "github.com/stackrox/rox/migrator/migrations/m_0_to_m_1_create_version_bucket"
	_ "github.com/stackrox/rox/migrator/migrations/m_1_to_2_alert_violation"
	_ "github.com/stackrox/rox/migrator/migrations/m_2_to_3_network_flows_in_badger"
)

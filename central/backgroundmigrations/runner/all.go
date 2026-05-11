package runner

// Import background migration packages here to register them via init().
import (
	_ "github.com/stackrox/rox/central/backgroundmigrations/migrations/m_000_to_m_001_bg_add_deployment_type_and_enforcement_count_to_alerts"
	_ "github.com/stackrox/rox/central/backgroundmigrations/migrations/m_001_to_m_002_bg_add_container_start_column_to_indicators"
	_ "github.com/stackrox/rox/central/backgroundmigrations/migrations/m_002_to_m_003_bg_add_updated_at_to_network_flows_v2"
)

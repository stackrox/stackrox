package runner

// Import background migration packages here to register them via init().
import (
	_ "github.com/stackrox/rox/central/backgroundmigrations/migrations/m_000_to_m_001_clear_base_image_tag_digests"
)

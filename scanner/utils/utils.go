package utils

// Claircore default vuln updaters
func GetCcUpdaters() []string {
	return []string{
		"alpine",
		"aws",
		"debian",
		"oracle",
		"osv",
		"photon",
		"rhel-vex",
		"suse",
		"ubuntu",
	}
}

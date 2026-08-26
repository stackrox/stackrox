package virtualmachine

// Facts keys used in VirtualMachine.Facts maps.
// Keep the keys camelCase to match the style used elsewhere in the UI.
const (
	GuestOSKey     = "guestOS"
	DescriptionKey = "description"
	IPAddressesKey = "ipAddresses"
	ActivePodsKey  = "activePods"
	NodeNameKey    = "nodeName"
	BootOrderKey   = "bootOrder"
	CDRomDisksKey  = "cdRomDisks"
	// DetectedGuestOSKey is the guest OS reported by roxagent, kept separate
	// from GuestOSKey so KubeVirt informer data is not overwritten.
	DetectedGuestOSKey = "detectedGuestOS"
	// ActivationStatusKey is whether the guest OS is activated.
	ActivationStatusKey = "activationStatus"
	// DNFMetadataStatusKey is whether DNF metadata is available on the guest.
	DNFMetadataStatusKey = "dnfMetadataStatus"
	// UnknownGuestOS is the user-facing default value for GuestOSKey when the
	// guest OS has not been reported by the virtual machine instance.
	UnknownGuestOS = "unknown"
	// User-facing values for roxagent-derived facts. Unspecified proto enum
	// values are omitted rather than stored as raw proto names.
	ActivationStatusActive       = "active"
	ActivationStatusInactive     = "inactive"
	DNFMetadataStatusAvailable   = "available"
	DNFMetadataStatusUnavailable = "unavailable"
)

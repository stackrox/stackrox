package virtualmachine

// Facts keys used in VirtualMachine.Facts maps.
// Keep the keys camelCase to match the style used elsewhere in the UI.
const (
	// GuestOSKey is the key for the guest OS of the virtual machine.
	GuestOSKey     = "guestOS"
	DescriptionKey = "description"
	IPAddressesKey = "ipAddresses"
	ActivePodsKey  = "activePods"
	NodeNameKey    = "nodeName"
	BootOrderKey   = "bootOrder"
	CDRomDisksKey  = "cdRomDisks"
	// roxagent-derived facts.
	// DetectedGuestOSKey is the key for the detected guest OS by the roxagent  running in the virtual machine.
	DetectedGuestOSKey = "detectedGuestOS"
	// ActivationStatusKey is the key for the activation status of the virtual machine.
	ActivationStatusKey = "activationStatus"
	// DNFMetadataStatusKey is the key for the DNF metadata status of the virtual machine.
	DNFMetadataStatusKey = "dnfMetadataStatus"
	// UnknownGuestOS is the user-facing default value for GuestOSKey when the
	// guest OS has not been reported by the virtual machine instance.
	UnknownGuestOS = "unknown"
	// User-facing values for roxagent-derived facts.
	ActivationStatusActive       = "active"
	ActivationStatusInactive     = "inactive"
	DNFMetadataStatusAvailable   = "available"
	DNFMetadataStatusUnavailable = "unavailable"
)

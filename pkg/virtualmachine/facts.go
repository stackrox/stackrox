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
	// UnknownGuestOS is the user-facing default value for GuestOSKey when the
	// guest OS has not been reported by the virtual machine instance.
	UnknownGuestOS = "unknown"
)

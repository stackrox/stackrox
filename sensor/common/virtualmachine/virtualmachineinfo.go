package virtualmachine

type VMID string

// Info information about a VirtualMachine
type Info struct {
	ID        VMID
	Name      string
	Namespace string
	VSOCKCID  *uint32
	Running   bool
	GuestOS   string
	// Description is derived from VM/VMI annotations (e.g. "description").
	Description string
	// IPAddresses are derived from VMI interface status.
	IPAddresses []string
	// ActivePods are derived from VMI status and formatted as "uid=node".
	ActivePods []string
	// NodeName is derived from VMI status.
	NodeName string
	// BootOrder contains disk boot order entries formatted as "disk=order".
	BootOrder []string
	// CDRomDisks lists disk names with CD-ROM devices.
	CDRomDisks []string
}

// Copy returns a copy of the VirtualMachineInfo
func (v *Info) Copy() *Info {
	if v == nil {
		return nil
	}
	ret := &Info{
		ID:        v.ID,
		Name:      v.Name,
		Namespace: v.Namespace,
		Running:   v.Running,
		GuestOS:   v.GuestOS,
		// Copy slices to avoid aliasing stored state.
		Description: v.Description,
		NodeName:    v.NodeName,
	}
	if v.VSOCKCID != nil {
		vsockCIDValue := *v.VSOCKCID
		ret.VSOCKCID = &vsockCIDValue
	}
	if len(v.IPAddresses) > 0 {
		ret.IPAddresses = append([]string(nil), v.IPAddresses...)
	}
	if len(v.ActivePods) > 0 {
		ret.ActivePods = append([]string(nil), v.ActivePods...)
	}
	if len(v.BootOrder) > 0 {
		ret.BootOrder = append([]string(nil), v.BootOrder...)
	}
	if len(v.CDRomDisks) > 0 {
		ret.CDRomDisks = append([]string(nil), v.CDRomDisks...)
	}
	return ret
}

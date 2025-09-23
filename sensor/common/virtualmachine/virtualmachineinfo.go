package virtualmachine

type VMID string

// Info information about a VirtualMachine
type Info struct {
	ID        VMID
	Name      string
	Namespace string
	VSOCKCID  *uint32
	Running   bool
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
	}
	if v.VSOCKCID != nil {
		vsockCIDValue := *v.VSOCKCID
		ret.VSOCKCID = &vsockCIDValue
	}
	return ret
}

package ipam

type CIDRManager interface {
	// ReserveCIDRs Reserve CIDRs on control plane reconstruction
	ReserveCIDRs([]string)
	// GetUnallocatedCIDR Get one unallocated CIDR
	GetUnallocatedCIDR() (string, error)
	// ReleaseCIDR Give up on using CIDR
	ReleaseCIDR(cidr string)
	// IsStatic Determines whether the allocator will always give back the same address
	IsStatic() bool
}

func NewIPAM(mode string) CIDRManager {
	switch mode {
	case "dynamic":
		return NewDynamicCIDRManager()
	default:
		return NewStaticCIDRManager(mode)
	}
}

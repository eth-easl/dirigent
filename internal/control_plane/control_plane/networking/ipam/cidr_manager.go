package ipam

type CIDRManager interface {
	ReserveCIDR() (string, error)
	ReleaseCIDR(cidr string)
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

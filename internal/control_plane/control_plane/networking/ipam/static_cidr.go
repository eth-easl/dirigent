package ipam

type StaticCIDRManager struct {
	cidr string
}

func NewStaticCIDRManager(cidr string) *StaticCIDRManager {
	return &StaticCIDRManager{
		cidr: cidr,
	}
}

func (s *StaticCIDRManager) ReserveCIDR() (string, error) {
	return s.cidr, nil
}

func (s *StaticCIDRManager) ReleaseCIDR(_ string) {}

func (s *StaticCIDRManager) IsStatic() bool {
	return true
}

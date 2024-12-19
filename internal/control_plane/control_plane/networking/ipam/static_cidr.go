package ipam

type StaticCIDRManager struct {
	cidr string
}

func NewStaticCIDRManager(cidr string) *StaticCIDRManager {
	return &StaticCIDRManager{
		cidr: cidr,
	}
}

func (s *StaticCIDRManager) ReserveCIDRs([]string) {}

func (s *StaticCIDRManager) GetUnallocatedCIDR() (string, error) {
	return s.cidr, nil
}

func (s *StaticCIDRManager) ReleaseCIDR(_ string) {}

func (s *StaticCIDRManager) IsStatic() bool {
	return true
}

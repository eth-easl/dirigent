package placement

type PlacementPolicy = int

const (
	PLACMENT_RANDOM PlacementPolicy = iota
	PLACEMENT_ROUND_ROBIN
	PLACEMENT_KUBERNETES
)

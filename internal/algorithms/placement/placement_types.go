package placement

type PlacementPolicy = int

const (
	RANDOM PlacementPolicy = iota
	ROUND_ROBIN
	KUBERNETES
)

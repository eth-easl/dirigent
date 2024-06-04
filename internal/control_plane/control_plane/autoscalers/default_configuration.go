package autoscalers

import (
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/proto"
	"math"
)

func NewDefaultAutoscalingConfiguration() *proto.AutoscalingConfiguration {
	return &proto.AutoscalingConfiguration{
		ScalingUpperBound:                    math.MaxInt32,
		ScalingLowerBound:                    0,
		PanicThresholdPercentage:             200,
		MaxScaleUpRate:                       1000.0,
		MaxScaleDownRate:                     2.0,
		ContainerConcurrency:                 1,
		ContainerConcurrencyTargetPercentage: 100,
		StableWindowWidthSeconds:             60,
		PanicWindowWidthSeconds:              6,
		ScalingPeriodSeconds:                 2,
		ScalingMethod:                        core.Arithmetic,
		TargetBurstCapacity:                  0,
		InitialScale:                         0,
		TotalValue:                           10000,
		ScaleDownDelay:                       2,
	}
}

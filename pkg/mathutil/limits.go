package mathutil

// This block declares numeric limit constants for the unsized integer types.
const (
	MaxUintVal = ^uint(0)
	MinUintVal = uint(0)

	MaxIntVal = int(MaxUintVal >> 1)
	MinIntVal = -MaxIntVal - 1
)

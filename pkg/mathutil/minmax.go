package mathutil

import "math"

// MinFloat32 takes the min
func MinFloat32(a, b float32) float32 {
	return float32(math.Min(float64(a), float64(b)))
}

// MinFloat64 takes the min
func MinFloat64(a, b float64) float64 {
	return math.Min(a, b)
}

// MinInt takes the min
func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// MinInt8 takes the min
func MinInt8(a, b int8) int8 {
	if a < b {
		return a
	}
	return b
}

// MinInt16 takes the min
func MinInt16(a, b int16) int16 {
	if a < b {
		return a
	}
	return b
}

// MinInt32 takes the min
func MinInt32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

// MinInt64 takes the min
func MinInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// MinUint takes the min
func MinUint(a, b uint) uint {
	if a < b {
		return a
	}
	return b
}

// MinUint8 takes the min
func MinUint8(a, b uint8) uint8 {
	if a < b {
		return a
	}
	return b
}

// MinUint16 takes the min
func MinUint16(a, b uint16) uint16 {
	if a < b {
		return a
	}
	return b
}

// MinUint32 takes the min
func MinUint32(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

// MinUint64 takes the min
func MinUint64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

// Max funcs

// MaxFloat32 takes the max
func MaxFloat32(a, b float32) float32 {
	return float32(math.Max(float64(a), float64(b)))
}

// MaxFloat64 takes the max
func MaxFloat64(a, b float64) float64 {
	return math.Max(a, b)
}

// MaxInt takes the max
func MaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// MaxInt8 takes the max
func MaxInt8(a, b int8) int8 {
	if a > b {
		return a
	}
	return b
}

// MaxInt16 takes the max
func MaxInt16(a, b int16) int16 {
	if a > b {
		return a
	}
	return b
}

// MaxInt32 takes the max
func MaxInt32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

// MaxInt64 takes the max
func MaxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// MaxUint takes the max
func MaxUint(a, b uint) uint {
	if a > b {
		return a
	}
	return b
}

// MaxUint8 takes the max
func MaxUint8(a, b uint8) uint8 {
	if a > b {
		return a
	}
	return b
}

// MaxUint16 takes the max
func MaxUint16(a, b uint16) uint16 {
	if a > b {
		return a
	}
	return b
}

// MaxUint32 takes the max
func MaxUint32(a, b uint32) uint32 {
	if a > b {
		return a
	}
	return b
}

// MaxUint64 takes the max
func MaxUint64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

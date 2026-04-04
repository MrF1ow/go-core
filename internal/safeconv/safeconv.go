package safeconv

import "math"

// ToInt32 safely converts int to int32, clamping to [MinInt32, MaxInt32].
func ToInt32(v int) int32 {
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	if v < math.MinInt32 {
		return math.MinInt32
	}
	return int32(v) // #nosec G115
}

// Uint32ToInt32 safely converts uint32 to int32, clamping to MaxInt32.
func Uint32ToInt32(v uint32) int32 {
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	return int32(v) // #nosec G115
}

// Int32ToUint32 safely converts int32 to uint32, clamping negative values to 0.
func Int32ToUint32(v int32) uint32 {
	if v < 0 {
		return 0
	}
	return uint32(v) // #nosec G115
}

// Uint64ToInt safely converts uint64 to int, clamping to MaxInt.
func Uint64ToInt(v uint64) int {
	if v > math.MaxInt {
		return math.MaxInt
	}
	return int(v) // #nosec G115
}

// IntToUint64 safely converts int to uint64, clamping negative values to 0.
func IntToUint64(v int) uint64 {
	if v < 0 {
		return 0
	}
	return uint64(v) // #nosec G115
}

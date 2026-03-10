package service

import "hash/fnv"

// hashBytes computes a 64-bit FNV-1a hash of the given byte slice.
// Returns 0 for nil/empty input.
func hashBytes(data []byte) uint64 {
	if len(data) == 0 {
		return 0
	}
	h := fnv.New64a()
	h.Write(data)
	return h.Sum64()
}

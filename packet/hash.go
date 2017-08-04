package packet

func fnvHash(s []byte) (h uint64) {
	h = fnvBasis
	for _, b := range s {
		h ^= uint64(b)
		h *= fnvPrime
	}
	return
}

const fnvBasis = 14695981039346656037
const fnvPrime = 1099511628211

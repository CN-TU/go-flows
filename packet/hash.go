package packet

func fnvHash(s []byte) (h uint64) {
	h = fnvBasis
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= fnvPrime
	}
	return
}

const fnvBasis = 14695981039346656037
const fnvPrime = 1099511628211

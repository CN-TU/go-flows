package features

// BoolInt returns 1 if b is true, otherwise 0
func BoolInt(b bool) uint64 {
	if b {
		return 1
	}
	return 0
}

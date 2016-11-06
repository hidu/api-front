package utils

// InInt64Slice check int64 in slice
func InInt64Slice(id int64, idSli []int64) bool {
	for _, v := range idSli {
		if id == v {
			return true
		}
	}
	return false
}

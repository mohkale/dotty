package pkg

func StringSliceContains(strs []string, val string) bool {
	for _, key := range strs {
		if key == val {
			return true
		}
	}

	return false
}

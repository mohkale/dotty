package main

func stringSliceContains(strs []string, val string) bool {
	for _, key := range strs {
		if key == val {
			return true
		}
	}

	return false
}

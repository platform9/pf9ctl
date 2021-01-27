package util

// Intersect returns a list of common items between two lists
func Intersect(a []string, b []string) []string {
	set := make([]string, 0)
	hash := make(map[string]bool)

	for i := 0; i < len(a); i++ {
		el := a[i]
		hash[el] = true
	}

	for i := 0; i < len(b); i++ {
		el := b[i]
		if _, found := hash[el]; found {
			set = append(set, el)
		}
	}

	return set
}

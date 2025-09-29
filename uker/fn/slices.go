package fn

// ContainsInSlice reports whether the provided value exists in the slice.
func ContainsInSlice[T comparable](slice []T, value T) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}

	return false
}

// Map applies the provided function to every element in the slice.
func Map[T any, R any](values []T, mapper func(T) R) []R {
	result := make([]R, 0, len(values))
	for _, value := range values {
		result = append(result, mapper(value))
	}

	return result
}

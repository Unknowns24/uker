package fn

// Ptr returns a pointer to the provided value.
func Ptr[T any](value T) *T {
	return &value
}

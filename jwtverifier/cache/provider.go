package cache

type Provider interface {
	// Get returns the value associated with the given key.
	Get(string) (interface{}, error)
}

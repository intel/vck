package reconcile

type lifecycle string

const (
	exists       lifecycle = "Exists"
	doesNotExist lifecycle = "Does-not-exist"
	deleting     lifecycle = "Deleting"
)

// isOneOf returns true if this lifecycle is in the supplied list.
func (l lifecycle) isOneOf(targets ...lifecycle) bool {
	for _, target := range targets {
		if l == target {
			return true
		}
	}
	return false
}

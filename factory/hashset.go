package factory

type HashSet[T comparable] struct {
	elements map[T]struct{}
}

// NewHashSet initializes and returns a new generic set
func NewHashSet[T comparable]() *HashSet[T] {
	return &HashSet[T]{
		elements: make(map[T]struct{}),
	}
}

// Add inserts an item into the set
func (s *HashSet[T]) Add(item T) {
	s.elements[item] = struct{}{}
}

// Remove deletes an item from the set
func (s *HashSet[T]) Remove(item T) {
	delete(s.elements, item)
}

// Contains checks if an item exists in the set
func (s *HashSet[T]) Contains(item T) bool {
	_, exists := s.elements[item]
	return exists
}

/*
// Items returns a slice of all items in the set
func (s *HashSet) Items() []string {
	items := make([]string, 0, len(s.elements))
	for item := range s.elements {
		items = append(items, item)
	}
	return items
}*/

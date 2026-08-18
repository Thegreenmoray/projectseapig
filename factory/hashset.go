type HashSet struct {
	elements map[string]struct{}
}

// NewHashSet initializes and returns a new set
func NewHashSet() *HashSet {
	return &HashSet{
		elements: make(map[string]struct{}),
	}
}

// Add inserts an item into the set
func (s *HashSet) Add(item string) {
	s.elements[item] = struct{}{}
}

// Remove deletes an item from the set
func (s *HashSet) Remove(item string) {
	delete(s.elements, item)
}

// Contains checks if an item exists in the set
func (s *HashSet) Contains(item string) bool {
	_, exists := s.elements[item]
	return exists
}

// Size returns the number of items in the set
func (s *HashSet) Size() int {
	return len(s.elements)
}

// Items returns a slice of all items in the set
func (s *HashSet) Items() []string {
	items := make([]string, 0, len(s.elements))
	for item := range s.elements {
		items = append(items, item)
	}
	return items
}
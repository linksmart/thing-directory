package simplexml

// Search is a slice of *Tag
type Search []*Tag

// ByName searches through the children Tags of each element in Search looking for case sensitive matches of Name and returns a new Search of the results. Namespace is ignored.
func (se Search) ByName(s string) Search {
	var r Search

	for _, v := range se {
		for _, v2 := range v.Tags() {
			if v2.Name == s {
				r = append(r, v2)
			}
		}
	}

	return r
}

// One returns the top result off of a Search
func (se Search) One() *Tag {
	if len(se) > 0 {
		return se[0]
	}
	return nil
}

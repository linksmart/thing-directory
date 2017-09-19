package simplexml

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Tag is an Element that can contain multiple child Elements
type Tag struct {
	// Name is the name of the current XML Element
	Name string

	// Prefix is the optional prefix of an XML element (eg. <prefix:name>)
	Prefix string

	// Attributes is a slice of *Attribute. Adding, reordering and removing Attributes may be done directly to this slice or with helper methods
	Attributes []*Attribute

	// elements is a slice of interface, gaurenteed to be a pointer through AddBefore and AddAfter
	elements []Element

	// parents is a slice of *Tag in order of parent to child relationship. It is used to build an XPath, determine
	// indent depth for MarshalIndent() and find available namespaces.
	parents []*Tag

	// formatPrefix is the prefix value used during MarshalIndent
	formatPrefix string

	// formatIndent is the indent value used during MarshalIndent
	formatIndent string
}

// setIndent recursively sets a Tags prefix and indent values for use during MarshalIndent
func (t *Tag) setIndent(indent string, prefix string) {
	t.formatPrefix = prefix
	t.formatIndent = indent
	for _, v := range t.Tags() {
		v.setIndent(prefix, indent)
	}
}

// AddBefore takes an Element pointer (add) and an optional Element pointer (before).
// If before == nil, the add element will be prepended to the elements slice, otherwise it will be placed
// before the 'before' element. If 'before' != nil and is not found in the current Tags elements, an error
// will be returned. This method is not recursive.
func (t *Tag) AddBefore(add Element, before Element) error {
	// verify add is pointer
	if reflect.ValueOf(add).Kind() != reflect.Ptr {
		return errors.New("non-pointer 'add' passed to AddBefore")
	}

	if before != nil {
		// verify before is pointer
		refBefore := reflect.ValueOf(before)
		if refBefore.Kind() != reflect.Ptr {
			return errors.New("non-pointer 'before' passed to AddBefore")
		}

		// find address in current tag
		loc := -1
		for k, v := range t.elements {
			if refBefore.Pointer() == reflect.ValueOf(v).Pointer() {
				loc = k
				break
			}
		}

		if loc >= 0 {
			// add the new element before the matched element in the slice
			t.elements = append(t.elements[:loc], append([]Element{add}, t.elements[loc:]...)...)
		} else {
			return errors.New("memory address of 'before' not in current Tag")
		}
	} else {
		// prepend to elements
		t.elements = append([]Element{add}, t.elements...)
	}

	return nil
}

// AddAfter takes an Element pointer (add) and an optional Element pointer (after).
// If before == nil, the add element will be appended to the elements slice, otherwise it will be placed
// after the 'after' element. If 'after' != nil and is not found in the current Tags elements, an error
// will be returned. This method is not recursive.
func (t *Tag) AddAfter(add Element, after Element) error {
	// verify add is pointer
	if reflect.ValueOf(add).Kind() != reflect.Ptr {
		return errors.New("non-pointer 'add' passed to AddBefore")
	}

	if after != nil {
		// verify after is pointer
		refAfter := reflect.ValueOf(after)
		if refAfter.Kind() != reflect.Ptr {
			return errors.New("non-pointer 'after' passed to AddBefore")
		}

		// find address in current tag
		loc := -1
		for k, v := range t.elements {
			if refAfter.Pointer() == reflect.ValueOf(v).Pointer() {
				loc = k
				break
			}
		}

		if loc >= 0 {
			// add the new element after the matched element in the slice
			t.elements = append(t.elements[:loc+1], append([]Element{add}, t.elements[loc+1:]...)...)
		} else {
			return errors.New("memory address of 'after' not in current Tag")
		}
	} else {
		// append to elements
		t.elements = append(t.elements, add)
	}

	return nil
}

// Remove will delete an element from the Tag based on the given memory address. An error will be returned if the
// memory address is not an element of the Tag. This function is not recursive.
func (t *Tag) Remove(remove Element) error {
	// verify remove is pointer
	refRemove := reflect.ValueOf(remove)
	if refRemove.Kind() != reflect.Ptr {
		return errors.New("non-pointer 'remove' passed to Remove")
	}

	// find address in current tag
	loc := -1
	for k, v := range t.elements {
		if refRemove.Pointer() == reflect.ValueOf(v).Pointer() {
			loc = k
			break
		}
	}

	if loc >= 0 {
		// delete the matched element from the slice
		t.elements = append(t.elements[:loc], t.elements[loc+1:]...)
	} else {
		return errors.New("memory address of 'remove' not in current Tag")
	}

	return nil
}

// Elements returns a slice of the Tags child Elements
func (t Tag) Elements() []Element {
	var s []Element

	for _, v := range t.elements {
		s = append(s, v)
	}

	return s
}

// Tags returns a slice of *Tag that are elements of the current Tag. This function is not recursive.
func (t Tag) Tags() []*Tag {
	var s []*Tag

	for _, v := range t.elements {
		switch t := v.(type) {
		case *Tag:
			s = append(s, t)
		}
	}

	return s
}

// AvailableNamespaces returns a slice of Attribute from the current Tag and it's
// parents in which IsNamespace() returns true
func (t Tag) AvailableNamespaces() []Attribute {
	namespaces := []Attribute{}

	// check current Tag's attributes
	for _, attr := range t.Attributes {
		if attr.IsNamespace() {
			namespaces = append(namespaces, *attr)
		}
	}

	// check parent attributes
	for _, pt := range t.parents {
		for _, attr := range pt.Attributes {
			if attr.IsNamespace() {
				namespaces = append(namespaces, *attr)
			}
		}
	}
	return namespaces
}

// GetPrefix iterates through AvailableNamespaces() and returns a prefix string for the given namespace. An error
// is returned upon 0 or more than 1 result.
func (t Tag) GetPrefix(ns string) (string, error) {
	var r []Attribute

	for _, attr := range t.AvailableNamespaces() {
		if attr.IsNamespace() && attr.Value == ns {
			r = append(r, attr)
		}
	}

	if len(r) == 1 {
		return r[0].Name, nil
	} else if len(r) > 1 {
		return "", errors.New(fmt.Sprintf("prefix for namespace '%s' defined more than once", ns))
	}

	return "", errors.New(fmt.Sprintf("prefix for namespace '%s' not available", ns))
}

// GetNamespace iterates through AvailableNamespaces() and returns a namespace string for the given prefix. An error
// is returned upon 0 or more than 1 result.
func (t Tag) GetNamespace(prefix string) (string, error) {
	var r []Attribute

	for _, attr := range t.AvailableNamespaces() {
		if attr.IsNamespace() && attr.Name == prefix {
			r = append(r, attr)
		}
	}

	if len(r) == 1 {
		return r[0].Value, nil
	} else if len(r) > 1 {
		return "", errors.New(fmt.Sprintf("namespace for prefix '%s' defined more than once", prefix))
	}

	return "", errors.New(fmt.Sprintf("namespace for prefix '%s' not available", prefix))
}

// AddAttribute appends a new Attribute to the Tag.
func (t *Tag) AddAttribute(name string, value string, prefix string) *Tag {
	t.Attributes = append(t.Attributes, &Attribute{prefix, name, value})
	return t
}

// AddNamespace is a wrapper for AddAttribute, setting the prefix to 'xmlns'.
func (t *Tag) AddNamespace(name string, value string) *Tag {
	t.Attributes = append(t.Attributes, &Attribute{"xmlns", name, value})
	return t
}

/*
TODO: Removed from scope of v0.1

// XPath returns the Tag's XPath from it's root
func (t Tag) XPath() XPath {
	x := XPath{}

	for _, v := range t.parents {
		x = append(x, v.Name)
	}

	return append(x, t.Name)
}
*/
// innerValue returns a string representation of the inner contents of a Tag
func (t Tag) innerValue() string {
	var s string

	for _, e := range t.elements {
		s = s + e.String()
	}

	return s
}

// Value returns the inner value of a non Comment and non Tag element. Value will return an
// empty string if there are 0 and an error if there are > 1.
func (t Tag) Value() (string, error) {
	var el []Element

	for _, v := range t.elements {
		switch ty := v.(type) {
		case *Tag:
		case *Comment:
		default:
			el = append(el, ty)
		}
	}

	if len(el) > 1 {
		return "", errors.New("multiple value type elements found in tag")
	} else if len(el) == 0 {
		return "", nil
	}

	return el[0].Value()
}

// String returns a string representation of the entire Tag and its inner contents. No error
// checking is done during String(), allowing for invalid XML to be produced.
func (t Tag) String() string {
	// TODO: method for indenting could be cleaner
	var tagIndent string
	var innerIndent string
	var trailingInnerIndent string
	var tagPrefix string
	var attr string

	for i := 0; i < len(t.parents); i++ {
		tagIndent = tagIndent + t.formatIndent
	}

	if (t.formatPrefix != "" || t.formatIndent != "") && len(t.Tags()) > 0 {
		innerIndent = "\n"
		trailingInnerIndent = innerIndent + t.formatPrefix + tagIndent
	}

	if t.Prefix != "" {
		tagPrefix = fmt.Sprintf("%s:", t.Prefix)
	}

	if len(t.Attributes) > 0 {
		var s []string
		for _, v := range t.Attributes {
			s = append(s, v.String())
		}
		attr = " " + strings.Join(s, " ")
	}

	v := t.innerValue()
	if v == "" {
		return fmt.Sprintf("%s%s<%s%s%s/>", t.formatPrefix, tagIndent, tagPrefix, t.Name, attr)
	}
	return fmt.Sprintf("%s%s<%s%s%s>%s%s%s</%s%s>", t.formatPrefix, tagIndent, tagPrefix, t.Name, attr, innerIndent, v, trailingInnerIndent, tagPrefix, t.Name)
}

// Marshal is a wrapper for String() but returns a []byte, error to conform to the normal Marshaler interface.
func (t *Tag) Marshal() ([]byte, error) {
	t.setIndent("", "")
	return []byte(t.String()), nil
}

/*
TODO: Removed from scope of v0.1

// UnmarshalStrict is a custom XML unmarshaller that behaves much like xml.Unmarshal with a few enahncements, including the return of a UnmarshalResult.
func (t Tag) UnmarshalStrict(v interface{}) (UnmarshalResult, error) {
	return UnmarshalResult{}, nil
}

// UnmarshalResult is returned during UnmarshalStrict(), containing a slice of XPath that were and were not successfully Unmarshaled.
type UnmarshalResult struct {
	Used   []XPath
	Unused []XPath
}

// Errors recursively checks the Tag for anything that makes the XML document invalid and returns a slice of error.
func (t Tag) Errors() []error {
	var errs []error

	// ensure a non Comment and Non Tag type do not coexist with a Tag type within the same Tag

			// scan all non Tag and non CDATA elements to determine if they should be CDATA
			for _, v := range t.elements {
				switch k := v.(type) {
				case *Tag:
					// skip
				case *CDATA:
					// skip
				default:
					fmt.Println(reflect.TypeOf(v).String())
					fmt.Println(v.String())
					if NeedCDATA(k.String()) {
						errs = append(errs, errors.New(t.XPath().String()+": should be CDATA"))
					}
				}
			}

		// ensure all CDATA elements do not contain ']]>'
		for _, v := range t.elements {
			switch k := v.(type) {
			case *CDATA:
				if strings.Contains(k.String(), "]]>") {
					errs = append(errs, errors.New(t.XPath().String()+": contains invalid CDATA"))
				}
			}
		}

		// ensure current prefix has a namespace
		if t.Prefix != "" {
			if _, err := t.GetNamespace(t.Prefix); err != nil {
				errs = append(errs, errors.New(t.XPath().String()+": prefix '"+t.Prefix+"' does not have a defined namespace"))
			}
		}

		// ensure all comments do not contain '--'
		for _, v := range t.elements {
			switch k := v.(type) {
			case *Comment:
				if strings.Contains(k.String(), "--") {
					errs = append(errs, errors.New(t.XPath().String()+": contains invalid Comments"))
				}
			}
		}

		// run on all child tags
		for _, v := range t.Tags() {
			errs = append(errs, v.Errors()...)
		}

	return errs
}
*/

// Search returns a new Search with the current Tag
func (t *Tag) Search() Search {
	return Search{t}
}

// NewTag returns a pointer to a new Tag with the given string
func NewTag(name string) *Tag {
	return &Tag{
		Name: name,
	}
}

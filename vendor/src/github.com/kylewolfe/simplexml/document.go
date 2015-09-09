// Package simplexml is a simple API to read, write, edit and search XML documents at run time in pure Go.
// A simplistic design relying on the fmt.Stringer interface to build a document.
package simplexml

import (
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"reflect"
	"strings"
)

type Document struct {
	Declaration string

	// elements is a slice of interface, gaurenteed to be a pointer through AddBefore and AddAfter
	elements []Element

	// formatPrefix is the prefix value used during MarshalIndent
	formatPrefix string

	// formatIndent is the indent value used during MarshalIndent
	formatIndent string
}

// Root returns a pointer to the Documents root element. Root will return an error if the
// Document does not contain exactly one *Tag
func (d Document) Root() *Tag {
	var r []*Tag

	for _, v := range d.elements {
		switch t := v.(type) {
		case *Tag:
			r = append(r, t)
		}
	}

	if len(r) == 0 {
		panic("The document does not contain a root element")
	} else if len(r) > 1 {
		panic("The document contains more than one root element")
	}

	return r[0]
}

// AddBefore takes an Element pointer (add) and an optional Element pointer (before).
// If before == nil, the add element will be prepended to the elements slice, otherwise it will be placed
// before the 'before' element. If 'before' != nil and is not found in the current Tags elements, an error
// will be returned. This method is not recursive.
func (d *Document) AddBefore(add Element, before Element) error {
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
		for k, v := range d.elements {
			if refBefore.Pointer() == reflect.ValueOf(v).Pointer() {
				loc = k
				break
			}
		}

		if loc >= 0 {
			// add the new element before the matched element in the slice
			d.elements = append(d.elements[:loc], append([]Element{add}, d.elements[loc:]...)...)
		} else {
			return errors.New("memory address of 'before' not in current Tag")
		}
	} else {
		// prepend to elements
		d.elements = append([]Element{add}, d.elements...)
	}

	return nil
}

// AddAfter takes an Element pointer (add) and an optional Element pointer (after).
// If before == nil, the add element will be appended to the elements slice, otherwise it will be placed
// after the 'after' element. If 'after' != nil and is not found in the current Tags elements, an error
// will be returned. This method is not recursive.
func (d *Document) AddAfter(add Element, after Element) error {
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
		for k, v := range d.elements {
			if refAfter.Pointer() == reflect.ValueOf(v).Pointer() {
				loc = k
				break
			}
		}

		if loc >= 0 {
			// add the new element after the matched element in the slice
			d.elements = append(d.elements[:loc+1], append([]Element{add}, d.elements[loc+1:]...)...)
		} else {
			return errors.New("memory address of 'after' not in current Tag")
		}
	} else {
		// append to elements
		d.elements = append(d.elements, add)
	}

	return nil
}

// Remove will delete an element from the Tag based on the given memory address. An error will be returned if the
// memory address is not an element of the Tag. This function is not recursive.
func (d *Document) Remove(remove Element) error {
	// verify remove is pointer
	refRemove := reflect.ValueOf(remove)
	if refRemove.Kind() != reflect.Ptr {
		return errors.New("non-pointer 'remove' passed to Remove")
	}

	// find address in current tag
	loc := -1
	for k, v := range d.elements {
		if refRemove.Pointer() == reflect.ValueOf(v).Pointer() {
			loc = k
			break
		}
	}

	if loc >= 0 {
		// delete the matched element from the slice
		d.elements = append(d.elements[:loc], d.elements[loc+1:]...)
	} else {
		return errors.New("memory address of 'remove' not in current Tag")
	}

	return nil
}

// setIndent sets teh indent for the current document and calls setIndent on its root element
func (d *Document) setIndent(indent string, prefix string) {
	d.formatPrefix = prefix
	d.formatIndent = indent
	d.Root().setIndent(prefix, indent)
}

// Marshal is a wrapper for String() but returns a []byte, error to conform to the normal Marshaler interface.
// An error will be returned if the doucment is malformed (returning the first result of Errors()).
func (d Document) Marshal() ([]byte, error) {
	d.setIndent("", "")

	/*
		TODO: Removed from scope of v0.1

		// return first error of root
		if errs := d.Root().Errors(); len(errs) != 0 {
			return nil, errs[0]
		}

	*/

	// build the document
	s := ""
	for _, v := range d.elements {
		s = s + v.String()
	}

	return []byte(s), nil
}

func NewDocument(t *Tag) *Document {
	return &Document{elements: []Element{t}}
}

// NewDocumentFromReader returns a new Document that is generated from an io.Reader using encoding/xml.Decoder
//
// BUG(kyle) Due to the design of xml.Decoder, Tags that define their namespace without a prefix will
// be converted to use the prefix if it was defined in a parent for use. The result will be a valid document,
// namespaces stay the same, just the resulting documents format is slightly different. This will be addressed once
// simplexml is no longer reliant on encoding/xml.
//
// eg: <foo xmlns:urn="http://foo"><bar xmlns="http://foo"/></foo> will be converted to
// <foo xmlns:urn="http://foo"><urn:bar xmlns="http://foo"/></foo>
func NewDocumentFromReader(r io.Reader) (*Document, error) {
	var start xml.StartElement
	var tree []*Tag
	var root *Tag

	doc := &Document{}

	d := xml.NewDecoder(r)
	for {
		tok, _ := d.Token()

		// done decoding on nil token
		if tok == nil {
			break
		}

		switch t := tok.(type) {
		case xml.StartElement:
			var tag *Tag
			start = t.Copy() // start is used through iterations

			// initiate root for first start element
			if len(tree) == 0 {
				root = &Tag{}
			}

			// set tag to be used, root if tree is empty, otherwise a new tag which is added to the tree
			if len(tree) == 0 {
				tag = root
				doc.elements = append(doc.elements, root)
			} else {
				tag = &Tag{parents: tree}
				tree[len(tree)-1].elements = append(tree[len(tree)-1].elements, tag)
			}

			// attributes need set so that available namespaces are updated
			for _, attr := range start.Attr {
				tag.AddAttribute(attr.Name.Local, attr.Value, attr.Name.Space)
			}

			// set tags name and prefix
			tag.Name = start.Name.Local
			prefix, err := tag.GetPrefix(start.Name.Space)
			if err == nil {
				tag.Prefix = prefix
			}

			// add new tag to the end of the tree
			tree = append(tree, tag)
		case xml.EndElement:
			// done with the element, drop it from working tree and reset start token
			tree = tree[:len(tree)-1]
			start = xml.StartElement{}
		case xml.CharData:
			// skip whitespace
			if strings.TrimSpace(string(t)) != "" {
				// decode the value
				v := html.UnescapeString(string(t))

				// add to latest element if elements in tree, otherwise add to doc
				if len(tree) > 0 {
					if NeedCDATA(v) {
						tree[len(tree)-1].AddAfter(NewCDATA(v), nil)
					} else {
						tree[len(tree)-1].AddAfter(NewValue(v), nil)
					}
				} else {
					if NeedCDATA(v) {
						doc.elements = append(doc.elements, NewCDATA(v))
					} else {
						doc.elements = append(doc.elements, NewValue(v))
					}

				}
			}
		case xml.Comment:
			// decode the value
			v := html.UnescapeString(string(t))

			// add to latest element if elements in tree, otherwise add to doc
			if len(tree) > 0 {
				tree[len(tree)-1].AddAfter(NewComment(v), nil)
			} else {
				doc.elements = append(doc.elements, NewComment(v))
			}
		case xml.ProcInst:
			doc.Declaration = fmt.Sprintf("<?xml %s?>", string(t.Inst))
		default:
			// eat token
		}
	}

	// we should be back down to the root tag
	if len(tree) != 0 {
		// TODO: position of failure
		return nil, errors.New("malformed document")
	}

	return doc, nil
}

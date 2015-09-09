package simplexml

import (
	"fmt"
	"html"
	"strings"
)

const (
	DefaultDeclaration = "<?xml version=\"1.0\" encding=\"UTF-8\"?>"
)

type Element interface {
	// String returns the elements string repesentation, including the elements wrapping markup, usually with HTML encoding
	String() string

	// Value returns the inner value of an Element, without the elements wrapping markup and without HTML encoding
	Value() (string, error)
}

// Value is a string representation of XML CharData
type Value string

// String implements the Stringer interface. String returns the html escaped value of Value.
func (v Value) String() string {
	return html.EscapeString(string(v))
}

// Value implements the Stringer interface. String returns the html escaped value of Value.
func (v Value) Value() (string, error) {
	return string(v), nil
}

// CDATA is a string representation of XML CDATA without the '<![CDATA[' and ']]>' markup.
type CDATA string

// String implements the Stringer interface. String returns the html escaped value of Value wrapped the CDATA markup.
func (c CDATA) String() string {
	return fmt.Sprintf("<![CDATA[%s]]>", html.EscapeString(string(c)))
}

// String implements the Stringer interface. String returns the html escaped value of Value wrapped the CDATA markup.
func (c CDATA) Value() (string, error) {
	return string(c), nil
}

// Comments is a string representation of an XML comment without the '<!--' and '-->' markup.
type Comment string

// String implements the Stringer interface. String returns the value of Comment with the comment markup. String()
// does not html encode the value, as it is not considered part of the document.
func (c Comment) String() string {
	return fmt.Sprintf("<!--%s-->", string(c))
}

func (c Comment) Value() (string, error) {
	return string(c), nil
}

// NewComment returns a pointer to a new Comment
func NewComment(s string) *Comment {
	c := new(Comment)
	*c = Comment(s)
	return c
}

// NewCDATA returns a pointer to a new CDATA
func NewCDATA(s string) *CDATA {
	c := new(CDATA)
	*c = CDATA(s)
	return c
}

// NewValue returns a pointer to a new CDATA
func NewValue(s string) *Value {
	c := new(Value)
	*c = Value(s)
	return c
}

// NeedCDATA parses a string and returns true if it contains any XML markup or other characters that would require it to be repesented as CDATA
func NeedCDATA(s string) bool {
	if strings.Contains(s, "<") || strings.Contains(s, ">") {
		return true
	}
	return false
}

// Attribute is a simple representations of an XML attrbiute, consiting of a prefix, name and value.
type Attribute struct {
	Prefix string
	Name   string
	Value  string
}

// IsNamespace returns true if it's prefix = 'xmlns' (not case sensitive)
func (a Attribute) IsNamespace() bool {
	if strings.ToLower(a.Prefix) == "xmlns" {
		return true
	}
	return false
}

// String returns a format for use within String() of Tag
func (a Attribute) String() string {
	if a.Prefix != "" {
		return fmt.Sprintf("%s:%s=\"%s\"", a.Prefix, a.Name, a.Value)
	}
	return fmt.Sprintf("%s=\"%s\"", a.Name, a.Value)
}

// XPath is a slice of string (of Tag names)
type XPath []string

// String returns the string repesenation of an XPATH ('/foo/bar')
func (x XPath) String() string {
	var s string
	for _, v := range x {
		s = s + "/" + v
	}
	return s
}

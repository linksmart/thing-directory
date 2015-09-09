simplexml [![Build Status](https://travis-ci.org/kylewolfe/simplexml.svg?branch=master)](https://travis-ci.org/kylewolfe/simplexml) [![Coverage Status](https://coveralls.io/repos/kylewolfe/simplexml/badge.svg)](https://coveralls.io/r/kylewolfe/simplexml) [![GoDoc](http://godoc.org/github.com/kylewolfe/simplexml?status.svg)](http://godoc.org/github.com/kylewolfe/simplexml) 
=========

simplexml provides a simple API to read, create and manipulate XML documents at run time in pure Go.

## Stability

simplxml underwent a major refactor for v0.1 in order to address comment support and a few other annoyances I had with the API. simplexml is now entering a more stable state as of v0.1. While trunk is not guaranteed to be clear of breaking changes, they will be well documented moving forward, and tags will be available of older releases. Please remember to vendor your dependencies :)

## Usage

### From Scratch

```go
root := NewTag("root")   // a tag is an element that can contain other elements
doc := NewDocument(root) // a document can only contain one root tag
doc.AddBefore(NewComment("simplexml has support for comments outside of the root document"), root)

root.AddAfter(NewTag("foo"), nil)  // a nil pointer can be given to append to the end of all elements
root.AddBefore(NewTag("bar"), nil) // or prepend before all elements

bat := NewTag("bat")
bat.AddAfter(NewValue("bat value"), nil)
root.AddAfter(bat, nil)

b, err := doc.Marshal() // a simplexml document implements the Marshaler interface
if err != nil {
	panic(err)
}
fmt.Println(string(b))
//Output:
//<!--simplexml has support for comments outside of the root document--><root><bar/><foo/><bat>bat value</bat></root>
```

### From A Reader
```go
xmlString := `<?xml version="1.0" encoding="UTF-8" standalone="no" ?>
<!-- comment above root element -->
<root>
<!-- <comment>above foo</comment> -->
<foo>
	<bar>bat</bar>
	<baz/>
	<fizz><![CDATA[&lt;foo&gt;contents&lt;/foo&gt;]]></fizz>
</foo>
</root>
<!-- comment below root element -->`

// create a document from a reader
doc, err := NewDocumentFromReader(strings.NewReader(xmlString))
if err != nil {
	panic(err)
}

// get the fizz tag and value
fizz := doc.Root().Search().ByName("foo").ByName("fizz").One()
if fizz == nil {
	panic("fizz is missing")
}

fv, err := fizz.Value()
if err != nil {
	panic(err)
}

fmt.Println("fizz: ", fv)
//Output:
//fizz:  <foo>contents</foo>
```
package simplexml

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"

	"fmt"
	"strings"
)

const (
	ExampleValidXML1 = `<?xml version="1.0" encoding="UTF-8" standalone="no" ?>
<!-- comment above root element -->
<root>
	<!-- <comment>above foo</comment> -->
	<foo>
		<bar>bat</bar>
		<baz/>
		<fizz><![CDATA[&lt;cdata&gt;contents&lt;/cdata&gt;]]></fizz>
	</foo>
</root>
<!-- comment below root element -->`
)

func TestRoot(t *testing.T) {
	Convey("Given a blank Document", t, func() {
		d := &Document{}

		Convey("Root() should panic", func() {
			So(func() { d.Root() }, ShouldPanic)
		})

		Convey("Given a new element of foo to the document", func() {
			d.elements = []Element{&Tag{Name: "foo"}}
			So(len(d.elements), ShouldEqual, 1)

			Convey("Root() should not be nil", func() {
				So(d.Root(), ShouldNotBeNil)
			})

			Convey("Given a second element of bar to the document", func() {
				d.elements = append(d.elements, &Tag{Name: "bar"})
				So(len(d.elements), ShouldEqual, 2)

				Convey("Root() should panic", func() {
					So(func() { d.Root() }, ShouldPanic)
				})
			})
		})
	})
}

func TestAddBefore(t *testing.T) {
	Convey("Given a Document with a root element of foo", t, func() {
		foo := &Tag{Name: "foo"}
		d := &Document{elements: []Element{foo}}
		So(len(d.elements), ShouldEqual, 1)

		Convey("AddBefore with a new Comment and a nil before pointer should prepend the element", func() {
			c := new(Comment)
			err := d.AddBefore(c, nil)
			So(err, ShouldBeNil)
			So(len(d.elements), ShouldEqual, 2)
			So(d.elements[0], ShouldHaveSameTypeAs, new(Comment))

			Convey("AddBefore with a new Value and a before pointer of the foo element, the Value element should be the second element of the document", func() {
				v := new(Value)
				err := d.AddBefore(v, foo)
				So(err, ShouldBeNil)
				So(len(d.elements), ShouldEqual, 3)
				So(d.elements[1], ShouldHaveSameTypeAs, new(Value))
			})
		})

		Convey("AddBefore with a new Element and a before pointer of an element not in the document should return an error", func() {
			err := d.AddBefore(new(Value), new(Value))
			So(err, ShouldNotBeNil)
		})
	})
}

func TestAddAfter(t *testing.T) {
	Convey("Given a Document with a root element of foo", t, func() {
		foo := &Tag{Name: "foo"}
		d := &Document{elements: []Element{foo}}
		So(len(d.elements), ShouldEqual, 1)

		Convey("AddAfter with a new Comment and a nil after pointer should append the element", func() {
			c := new(Comment)
			err := d.AddAfter(c, nil)
			So(err, ShouldBeNil)
			So(len(d.elements), ShouldEqual, 2)
			So(d.elements[1], ShouldHaveSameTypeAs, new(Comment))

			Convey("AddAfter with a new Value and a after pointer of the foo element, the Value element should be the second element of the document", func() {
				v := new(Value)
				err := d.AddAfter(v, foo)
				So(err, ShouldBeNil)
				So(len(d.elements), ShouldEqual, 3)
				So(d.elements[1], ShouldHaveSameTypeAs, new(Value))
			})
		})

		Convey("AddAfter with a new Element and a after pointer of an element not in the document should return an error", func() {
			err := d.AddAfter(new(Value), new(Value))
			So(err, ShouldNotBeNil)
		})
	})
}

func TestRemove(t *testing.T) {
	Convey("Given a Document with a root element of foo and surrounding comments", t, func() {
		foo := &Tag{Name: "foo"}
		d := &Document{elements: []Element{new(Comment), foo, new(Comment)}}
		So(len(d.elements), ShouldEqual, 3)

		Convey("Remove(foo) should remove the middle element, leaving two comment types", func() {
			err := d.Remove(foo)
			So(err, ShouldBeNil)
			So(len(d.elements), ShouldEqual, 2)
			So(d.elements[0], ShouldHaveSameTypeAs, new(Comment))
			So(d.elements[1], ShouldHaveSameTypeAs, new(Comment))
		})
	})
}

func TestNewDocument(t *testing.T) {
	Convey("Given a new document from NewDocument(&Tag{Name: \"foo\"})", t, func() {
		d := &Document{elements: []Element{&Tag{Name: "foo"}}}

		Convey("Its declaration should be empty", func() {
			So(d.Declaration, ShouldBeEmpty)
		})

		Convey("It should have one element", func() {
			So(len(d.elements), ShouldEqual, 1)
		})
	})
}

func TestNewDocumentFromReader(t *testing.T) {
	Convey("Given NewDocumentFromReader from ExampleValidXML1", t, func() {
		d, err := NewDocumentFromReader(strings.NewReader(ExampleValidXML1))
		So(err, ShouldBeNil)

		Convey("The document should have a 'root' element wrapped with two comment elements", func() {
			So(len(d.elements), ShouldEqual, 3)
			So(d.elements[0], ShouldHaveSameTypeAs, new(Comment))
			So(d.elements[1], ShouldHaveSameTypeAs, new(Tag))
			So(d.elements[2], ShouldHaveSameTypeAs, new(Comment))
			So(d.elements[1].(*Tag).Name, ShouldEqual, "root")
			root := d.elements[1].(*Tag)

			Convey("The root element should have a comment element and a tag of foo", func() {
				So(len(root.elements), ShouldEqual, 2)
				So(root.elements[0], ShouldHaveSameTypeAs, new(Comment))
				So(root.elements[1], ShouldHaveSameTypeAs, new(Tag))
				So(root.elements[1].(*Tag).Name, ShouldEqual, "foo")
				foo := root.elements[1].(*Tag)

				Convey("foo should have 3 Tag elements", func() {
					So(len(foo.elements), ShouldEqual, 3)
					So(foo.elements[0], ShouldHaveSameTypeAs, new(Tag))
					So(foo.elements[1], ShouldHaveSameTypeAs, new(Tag))
					So(foo.elements[2], ShouldHaveSameTypeAs, new(Tag))

					Convey("The value of bar should equal 'bat'", func() {
						v, err := foo.elements[0].Value()
						So(err, ShouldBeNil)
						So(v, ShouldEqual, "bat")
					})

					Convey("The value of baz should be empty", func() {
						v, err := foo.elements[1].Value()
						So(err, ShouldBeNil)
						So(v, ShouldBeEmpty)
					})

					Convey("The value of fizz should equal '<cdata>contents</cdata>'", func() {
						v, err := foo.elements[2].Value()
						So(err, ShouldBeNil)
						So(v, ShouldEqual, "<cdata>contents</cdata>")
					})

					Convey("The fizz element should have 1 CDATA element", func() {
						So(len(foo.elements[2].(*Tag).elements), ShouldEqual, 1)
						So(foo.elements[2].(*Tag).elements[0], ShouldHaveSameTypeAs, new(CDATA))
					})
				})
			})
		})

		Convey("The declaration should be the same as the string version", func() {
			So(d.Declaration, ShouldEqual, "<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"no\" ?>")
		})
	})
}

func TestMarshal(t *testing.T) {
	Convey("Given the result of Marshal from ExampleValidXML1", t, func() {
		d, err := NewDocumentFromReader(strings.NewReader(ExampleValidXML1))
		So(err, ShouldBeNil)
		b, err := d.Marshal()

		Convey("Error should be nil", func() {
			So(err, ShouldBeNil)
		})

		Convey("Bytes should not be nil", func() {
			So(b, ShouldNotBeNil)
		})

		Convey("String representation should equal the original document less whitespace", func() {
			So(string(b), ShouldEqual, "<!-- comment above root element --><root><!-- <comment>above foo</comment> --><foo><bar>bat</bar><baz/><fizz><![CDATA[&lt;cdata&gt;contents&lt;/cdata&gt;]]></fizz></foo></root><!-- comment below root element -->")
		})
	})
}

func ExampleNewDocument() {
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
}

func ExampleNewDocumentFromReader() {
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
}

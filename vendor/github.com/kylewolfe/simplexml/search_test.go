package simplexml

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestByName(t *testing.T) {
	Convey("Given a simple document with multiple children and varying depths", t, func() {
		root := NewTag("root")

		foo := NewTag("foo")
		foo_0 := NewTag("foo_0")
		foo_0.AddAfter(NewTag("foo_0_0"), nil)
		foo_0.AddAfter(NewTag("foo_0_1"), nil)
		foo_0.AddAfter(NewTag("foo_0_1"), nil)
		foo.AddAfter(foo_0, nil)

		foo_1 := NewTag("foo_1")
		foo_1.AddAfter(NewTag("foo_1_0"), nil)
		foo_1.AddAfter(NewTag("foo_1_1"), nil)
		foo.AddAfter(foo_1, nil)

		root.AddAfter(foo, nil)
		root.AddAfter(NewTag("bar"), nil)

		s := Search{root}

		Convey("ByName(\"foo\") on root should return 1 result", func() {
			So(len(s.ByName("foo")), ShouldEqual, 1)
		})

		Convey("ByName(\"foo\").ByName(\"foo_0\") should return 1 result", func() {
			So(len(s.ByName("foo").ByName("foo_0")), ShouldEqual, 1)
		})

		Convey("ByName(\"foo\").ByName(\"foo_0\").ByName(\"foo_0_1\") should return 2 results", func() {
			So(len(s.ByName("foo").ByName("foo_0").ByName("foo_0_1")), ShouldEqual, 2)
		})
	})
}

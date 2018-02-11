package goautodoc

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestRegexp(t *testing.T) {
	assert.True(t, signatureRe.MatchString("// > Test.foo()"))
	assert.True(t, commentLine.MatchString("// this is a test"))
	assert.True(t, startExample.MatchString("/* >"))
	assert.True(t, endExample.MatchString("*/"))
	assert.True(t, getHeader.MatchString("Test.foo()"))
}

var (
	testStr = `
// > sum(a,b,c)
// Sums the three values it's given.
// Note that all three values are required.
/* >
  var triplets = [
  	[1,2,3],
  	[3,1,4]
  ];
  var i,a,b,c;
  for (i=0; i<triplets.length;i++){
  	a = triplets[0];
  	b = triplets[1];
  	c = triplets[2];
  	console.log(sum(a,b,c));
  }
*/
function sum(a,b,c){
	return a+b+c;
}
`
	testStrExpected = `## test/autodoc<a name="__top"></a>

<sub><sup>[&larr;Home](index.md)</sup></sub>

* [sum](#sum)

### <a name='sum'></a>sum
~javascript
sum(a,b,c)
~
Sums the three values it's given.
Note that all three values are required.

<sub><sup>[&uarr;Top](#__top)</sup></sub>`
)

func TestDoc(t *testing.T) {
	in := bytes.NewBufferString(testStr)
	out := &bytes.Buffer{}
	assert.NoError(t, Document("test/autodoc", in, out))
	expected := strings.Replace(testStrExpected, "~", "```", -1)
	assert.Equal(t, expected, out.String())
}

var expectedIndex = `## Index of test/

<sub><sup>[Back](../index.md)</sup></sub>

* [a/](a//index.md)
* [b/](b//index.md)
* [bar.js](bar.js.md)
* [foo.js](foo.js.md)`

func TestIndexDoc(t *testing.T) {
	buf := &bytes.Buffer{}
	doc := indexDoc{
		title: "test/",
		files: []string{
			"foo.js",
			"bar.js",
		},
		dirs: []string{
			"b/",
			"a/",
		},
		Writer: buf,
	}
	doc.writeAll()
	assert.Equal(t, expectedIndex, buf.String())
}

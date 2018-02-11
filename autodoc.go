package goautodoc

import (
	"bufio"
	"io"
	"regexp"
	"sort"
	"strings"
)

var (
	signatureRe       = regexp.MustCompile(`^\s*//\s*>\s*(.*)`)
	commentLine       = regexp.MustCompile(`^\s*//\s*(.*)`)
	startExample      = regexp.MustCompile(`^\s*/\*\s*>`)
	endExample        = regexp.MustCompile(`^\s*\*/`)
	leadingWhitespace = regexp.MustCompile(`^[\t ]*`)
	getHeader         = regexp.MustCompile(`^[a-zA-Z0-9$._:-]*`)
	topLink           = "\n<sub><sup>[&uarr;Top](#__top)</sup></sub>"
	topAnchor         = "<a name=\"__top\"></a>\n\n"
	homeLink          = "<sub><sup>[&larr;Home](index.md)</sup></sub>\n\n"
	startJSblock      = "```javascript"
	endJSblock        = "```"
)

type docOp struct {
	title    string
	sections []*docSec
	indexes  []string
	*bufio.Reader
	io.Writer
	line []byte
	err  error
}

type docSec struct {
	header string
	text   []string
}

func (sec *docSec) writeString(str string) {
	sec.text = append(sec.text, str)
}

func (sec *docSec) write(b []byte) {
	sec.writeString(string(b))
}

func Document(title string, in io.Reader, out io.Writer) error {
	op := &docOp{
		title:  title,
		Reader: bufio.NewReader(in),
		Writer: out,
	}
	for op.readline() == nil {
		if sig := signatureRe.FindSubmatch(op.line); len(sig) == 2 {
			op.getSec(sig)
		}
	}
	if op.err == io.EOF {
		op.err = nil
	}

	op.index()
	op.writeAll()

	return op.err
}

func (op *docOp) readline() error {
	if op.err != nil {
		return op.err
	}
	op.line, _, op.err = op.ReadLine()
	return op.err
}

func (op *docOp) getSec(signature [][]byte) {
	sec := &docSec{
		header: string(getHeader.Find(signature[1])),
		text: []string{
			"", // place holder for title line
			startJSblock,
			string(signature[1]),
		},
	}

	inCodeBlock := true
	for op.readline() == nil {
		if sig := signatureRe.FindSubmatch(op.line); len(sig) == 2 {
			if !inCodeBlock {
				sec.writeString(startJSblock)
				inCodeBlock = true
			}
			sec.write(sig[1])
			continue
		}

		if inCodeBlock {
			sec.writeString(endJSblock)
			inCodeBlock = false
		}

		if docLine := commentLine.FindSubmatch(op.line); len(docLine) == 2 {
			sec.write(docLine[1])
			continue
		}

		if example := startExample.FindSubmatch(op.line); len(example) == 2 {
			op.getExample(sec)
			continue
		}

		break
	}

	if inCodeBlock {
		sec.writeString(endJSblock)
	}

	op.sections = append(op.sections, sec)
}

func (op *docOp) getExample(sec *docSec) {
	sec.writeString(startJSblock)
	err := op.readline()
	lws := string(leadingWhitespace.Find(op.line))
	for ; err == nil; err = op.readline() {
		if endExample.Match(op.line) {
			break
		}
		line := string(op.line)
		if strings.HasPrefix(line, lws) {
			line = strings.Replace(line, lws, "", 1)
		}
		sec.writeString(line)
	}
	sec.writeString(endJSblock)
}

func (op *docOp) index() {
	sort.Slice(op.sections, func(i, j int) bool {
		return op.sections[i].header < op.sections[j].header
	})

	var indentStack []string
	for _, sec := range op.sections {
		safeKey := strings.Replace(sec.header, ":", "_", -1)
		indentStack = indentLevel(sec.header, indentStack)
		indentLevel := len(indentStack)

		indexLine := []string{strings.Repeat("  ", indentLevel-1), "* [", sec.header, "](#", safeKey, ")\n"}
		op.indexes = append(op.indexes, strings.Join(indexLine, ""))

		titleLine := []string{strings.Repeat("#", indentLevel+2), " <a name='", safeKey, "'></a>", sec.header}
		sec.text[0] = strings.Join(titleLine, "")

		sec.writeString(topLink)
	}
}

func indentLevel(header string, indentStack []string) []string {
	for ln := len(indentStack); ln > 0; ln = len(indentStack) {
		if strings.HasPrefix(header, indentStack[ln-1]) {
			break
		}
		indentStack = indentStack[:ln-1]
	}
	return append(indentStack, header)
}

func (op *docOp) writeString(str string) error {
	if op.err != nil {
		return op.err
	}
	_, op.err = op.Write([]byte(str))
	return op.err
}

func (op *docOp) writeAll() {
	op.writeString("## ")
	op.writeString(op.title)
	op.writeString(topAnchor)
	op.writeString(homeLink)

	for _, idxStr := range op.indexes {
		op.writeString(idxStr)
	}

	for _, sec := range op.sections {
		op.writeString("\n")
		op.writeString(strings.Join(sec.text, "\n"))
	}
}

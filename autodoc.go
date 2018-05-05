package goautodoc

import (
	"bufio"
	"io"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
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

type docData struct {
	title    string
	sections []*docSec
	indexes  []string
	err      error
}

type docReader struct {
	docData
	*bufio.Reader
	line []byte
}

type docWriter struct {
	docData
	io.Writer
	n int64
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

func Document(title string, in io.Reader) (io.WriterTo, error) {
	op := &docReader{
		docData: docData{
			title: title,
		},
		Reader: bufio.NewReader(in),
	}
	for op.readline() == nil {
		if sig := signatureRe.FindSubmatch(op.line); len(sig) == 2 {
			op.getSec(sig)
		}
	}
	if op.err == io.EOF {
		op.err = nil
	}
	if len(op.sections) == 0 {
		return nil, nil
	}

	op.index()
	if op.err != nil {
		return nil, op.err
	}

	return &docWriter{
		docData: op.docData,
	}, nil
}

func (op *docWriter) WriteTo(w io.Writer) (int64, error) {
	op.Writer = w
	return op.writeAll()
}

func (op *docReader) readline() error {
	if op.err != nil {
		return op.err
	}
	op.line, _, op.err = op.ReadLine()
	return op.err
}

func (op *docReader) getSec(signature [][]byte) {
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

		if startExample.Match(op.line) {
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

func (op *docReader) getExample(sec *docSec) {
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

func (op *docReader) index() {
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
		// foo.bar.baz is sub-section of foo.bar
		// but
		// foo.barge is not a sub-section foo.bar
		if prevHeader := indentStack[ln-1]; len(header) > len(prevHeader) && strings.HasPrefix(header, prevHeader) {
			// check that the rune after the prevHeader prefix is a 'mark' rune
			l := utf8.RuneCountInString(prevHeader)
			r := ([]rune(header))[l]
			if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
				break
			}
		}
		indentStack = indentStack[:ln-1]
	}
	return append(indentStack, header)
}

func (op *docWriter) writeString(str string) error {
	if op.err != nil {
		return op.err
	}
	n, err := op.Write([]byte(str))
	op.err = err
	op.n += int64(n)
	return op.err
}

func (op *docWriter) writeAll() (int64, error) {
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
	return op.n, op.err
}

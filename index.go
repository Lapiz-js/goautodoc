package goautodoc

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	back = "\n<sub><sup>[Back](../index.md)</sup></sub>\n"
)

func DocumentDirectories(title, docPath string, baseDirs ...string) error {
	rootDoc := &indexDoc{
		title:  title,
		isRoot: true,
	}
	for _, dir := range baseDirs {
		base := filepath.Base(dir)
		rootDoc.dirs = append(rootDoc.dirs, base)
		err := indexDir(title+"/"+base, filepath.Join(docPath, base), dir)
		if err != nil {
			return err
		}
	}

	err := os.MkdirAll(docPath, 0777)
	if err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(docPath, "index.md"))
	if err != nil {
		return err
	}
	rootDoc.Writer = f
	rootDoc.writeAll()
	return f.Close()
}

func indexDir(title, docDir, codeDir string) error {
	idx := &indexDoc{
		title: title,
	}

	f, err := os.Open(codeDir)
	if err != nil {
		return err
	}
	list, err := f.Readdir(-1)
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}

	for _, f := range list {
		name := f.Name()
		if f.IsDir() && name != "tests" {
			idx.dirs = append(idx.dirs, name)
		} else if strings.HasSuffix(name, ".js") {
			err = os.MkdirAll(docDir, 0777)
			if err != nil {
				return err
			}
			out, err := os.Create(filepath.Join(docDir, name+".md"))
			if err != nil {
				return err
			}
			in, err := os.Open(filepath.Join(codeDir, name))
			if err != nil {
				out.Close()
				return err
			}
			err = Document(title+"/"+name, in, out)
			if err != nil {
				return err
			}
			out.Close()
			in.Close()
			idx.files = append(idx.files, name)
		}
	}

	for _, sub := range idx.dirs {
		err := indexDir(title+"/"+sub, filepath.Join(docDir, sub), filepath.Join(codeDir, sub))
		if err != nil {
			return err
		}
	}

	err = os.MkdirAll(docDir, 0777)
	if err != nil {
		return err
	}
	f, err = os.Create(filepath.Join(docDir, "index.md"))
	if err != nil {
		return err
	}
	idx.Writer = f
	idx.writeAll()
	return f.Close()
}

type indexDoc struct {
	title  string
	files  []string
	dirs   []string
	isRoot bool
	err    error
	io.Writer
}

func (i *indexDoc) write(str string) error {
	if i.err != nil {
		return i.err
	}
	_, i.err = i.Write([]byte(str))
	return i.err
}

func (i *indexDoc) writeAll() {
	i.write("## Index of ")
	i.write(i.title)
	i.write("\n")
	if !i.isRoot {
		i.write(back)
	}
	sort.Strings(i.dirs)
	sort.Strings(i.files)
	for _, dir := range i.dirs {
		i.write("\n* [")
		i.write(dir)
		i.write("](")
		i.write(dir)
		i.write("/index.md)")
	}
	for _, file := range i.files {
		i.write("\n* [")
		i.write(file)
		i.write("](")
		i.write(file)
		i.write(".md)")
	}
}

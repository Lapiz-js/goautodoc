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
		ok, err := indexDir(title+"/"+base, filepath.Join(docPath, base), dir)
		if err != nil {
			return err
		}
		if ok {
			rootDoc.dirs = append(rootDoc.dirs, base)
		}
	}

	ok, err := exists(docPath)
	if err != nil {
		return err
	}
	if !ok {
		err := os.MkdirAll(docPath, 0777)
		if err != nil {
			return err
		}
	}
	f, err := os.Create(filepath.Join(docPath, "index.md"))
	if err != nil {
		return err
	}
	rootDoc.Writer = f
	rootDoc.writeAll()
	return f.Close()
}

func indexDir(title, docDir, codeDir string) (bool, error) {
	idx := &indexDoc{
		title: title,
	}

	f, err := os.Open(codeDir)
	if err != nil {
		return false, err
	}
	list, err := f.Readdir(-1)
	if err != nil {
		return false, err
	}
	err = f.Close()
	if err != nil {
		return false, err
	}

	var dirs []string
	for _, f := range list {
		name := f.Name()
		if f.IsDir() && name != "tests" {
			dirs = append(dirs, name)
		} else if strings.HasSuffix(name, ".js") {
			ok, err := documentFile(title, name, docDir, codeDir)
			if err != nil {
				return false, err
			}
			if ok {
				idx.files = append(idx.files, name)
			}
		}
	}

	for _, dir := range dirs {
		ok, err := indexDir(title+"/"+dir, filepath.Join(docDir, dir), filepath.Join(codeDir, dir))
		if err != nil {
			return false, err
		}
		if ok {
			idx.dirs = append(idx.dirs, dir)
		}
	}

	if len(idx.files) == 0 && len(idx.dirs) == 0 {
		return false, nil
	}

	ok, err := exists(docDir)
	if err != nil {
		return false, err
	}
	if !ok {
		err = os.MkdirAll(docDir, 0777)
		if err != nil {
			return false, err
		}
	}
	f, err = os.Create(filepath.Join(docDir, "index.md"))
	if err != nil {
		return false, err
	}
	idx.Writer = f
	idx.writeAll()
	return true, f.Close()
}

func documentFile(title, name, docDir, codeDir string) (bool, error) {
	in, err := os.Open(filepath.Join(codeDir, name))
	if err != nil {
		return false, err
	}
	defer in.Close()

	writer, err := Document(title+"/"+name, in)
	if err != nil || writer == nil {
		return false, err
	}

	ok, err := exists(docDir)
	if err != nil {
		return false, err
	}
	if !ok {
		err = os.MkdirAll(docDir, 0777)
		if err != nil {
			return false, err
		}
	}
	out, err := os.Create(filepath.Join(docDir, name+".md"))
	if err != nil {
		return false, err
	}
	defer out.Close()

	_, err = writer.WriteTo(out)
	return true, err
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

func exists(dir string) (bool, error) {
	_, err := os.Stat(dir)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

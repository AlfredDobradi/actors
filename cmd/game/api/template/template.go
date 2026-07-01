package template

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
)

//go:embed tpl
var files embed.FS

var cache map[string]*template.Template

func Render(name string, data any) ([]byte, error) {
	if cache == nil {
		cache = make(map[string]*template.Template)
	}

	filename := fmt.Sprintf("tpl/%s.gohtml", name)

	tpl, ok := cache[filename]
	if !ok {
		newTpl, err := template.ParseFS(files, filename)
		if err != nil {
			return nil, nil
		}

		tpl = newTpl
		cache[filename] = tpl
	}

	buf := bytes.NewBufferString("")
	if err := tpl.Execute(buf, data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

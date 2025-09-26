package utils

import (
	"bytes"
	"text/template"
	"time"
)

// MustExecuteTemplate 执行模板
func MustExecuteTemplate(t string, data any) string {
	var v bytes.Buffer

	funcMap := template.FuncMap{
		"now":   time.Now,                                                 // 当前时间
		"today": func() string { return time.Now().Format("2006-01-02") }, // 当前时间
	}

	tpl, err := template.New(`MustExecuteTemplate`).Funcs(funcMap).Parse(t)
	if err != nil {
		panic(err)
		//return t
	}
	if err = tpl.Execute(&v, data); err != nil {
		panic(err)
		//return t
	}
	return v.String()
}

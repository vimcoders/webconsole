package generator

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/tealeg/xlsx/v3"
)

func TestGenerator(t *testing.T) {
	g := &Generator{
		codeName: make(map[string]int32),
	}
	fileInfo, err := ioutil.ReadDir("../xlsx")
	if err != nil {
		t.Error(err)
		return
	}
	var codeName []*CodeName
	for _, f := range fileInfo {
		if ok := f.IsDir(); ok {
			continue
		}
		file, err := xlsx.OpenFile(fmt.Sprintf("../xlsx/%v", f.Name()))
		if err != nil {
			t.Error(err)
		}
		g.Sheets = append(g.Sheets, file.Sheets...)
		for _, v := range file.Sheets {
			codeName = append(codeName, g.CodeName(v)...)
		}
	}
	for _, v := range codeName {
		g.codeName[v.Name] = v.Code
	}
	for _, v := range g.Sheets {
		g.WriteTo(v, os.Stdout)
		t.Log(g.Unmarshal(v))
	}
	for _, v := range g.Sheets {
		http.HandleFunc("/api/conf", func(w http.ResponseWriter, r *http.Request) {
			b, err := g.Unmarshal(v)
			if err != nil {
				t.Log(err)
			}
			w.Write(b)
		})
	}
	http.ListenAndServe(":8081", nil)
}

func TestDownload(t *testing.T) {
}

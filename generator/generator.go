package generator

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/template"

	"github.com/tealeg/xlsx/v3"
)

func init() {
	http.HandleFunc("/api/conf/gen", func(w http.ResponseWriter, r *http.Request) {
		g := NewGenerator()
		if g == nil {
			return
		}
		hander := func(s *xlsx.Sheet) {
			p := fmt.Sprintf("/api/conf/%v", strings.ToLower(s.Name))
			http.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
				b, err := g.Unmarshal(s)
				if err != nil {
					w.Write([]byte(err.Error()))
					return
				}
				w.Write(b)
			})
		}
		for i := 0; i < len(g.Sheets); i++ {
			hander(g.Sheets[i])
		}
		if err := g.Gen(); err != nil {
			w.Write([]byte(err.Error()))
			return
		}
	})
}

func WriteResponse(w http.ResponseWriter, result interface{}) error {
	b, err := json.Marshal(result)
	if err != nil {
		return err
	}
	if _, err = w.Write(b); err != nil {
		return err
	}
	return nil
}

type CodeName struct {
	Name string
	Code int32
}

type Generator struct {
	Sheets   []*xlsx.Sheet
	codeName map[string]int32
}

func NewGenerator() *Generator {
	g := Generator{
		codeName: make(map[string]int32),
	}
	fileInfo, err := ioutil.ReadDir("./xlsx")
	if err != nil {
		return nil
	}
	var codeName []*CodeName
	for _, f := range fileInfo {
		if ok := f.IsDir(); ok {
			continue
		}
		file, err := xlsx.OpenFile(fmt.Sprintf("./xlsx/%v", f.Name()))
		if err != nil {
			return nil
		}
		g.Sheets = append(g.Sheets, file.Sheets...)
		for _, v := range g.Sheets {
			codeName = append(codeName, g.CodeName(v)...)
		}
	}
	for _, v := range codeName {
		g.codeName[v.Name] = v.Code
	}
	return &g
}

func (g *Generator) Convert(str string) string {
	switch {
	case strings.Contains(str, "_IDX"):
		return fmt.Sprintf("%v int32\n", str)
	case strings.Contains(str, "_CONST"):
		return fmt.Sprintf("%v int32\n", str)
	case strings.Contains(str, "_INT"):
		return fmt.Sprintf("%v int32\n", str)
	case strings.Contains(str, "_STR"):
		return fmt.Sprintf("%v string\n", str)
	}
	return ""
}

func (g *Generator) unmarshal(t, d *xlsx.Row) map[string]interface{} {
	m := make(map[string][]interface{})
	for i := 0; i < t.Sheet.MaxCol; i++ {
		tt := t.GetCell(i)
		dd := d.GetCell(i)
		switch {
		case strings.Contains(tt.Value, "_CONST"):
			if v, ok := g.codeName[dd.Value]; ok {
				m[tt.Value] = append(m[tt.Value], v)
				continue
			}
		case strings.Contains(tt.Value, "_IDX"):
		case strings.Contains(tt.Value, "_INT"):
			v, err := strconv.Atoi(dd.Value)
			if err != nil {
				panic(err)
				return nil
			}
			m[tt.Value] = append(m[tt.Value], v)
		case strings.Contains(tt.Value, "_STR"):
			m[tt.Value] = append(m[tt.Value], dd.Value)
		}
	}
	result := make(map[string]interface{})
	for k, v := range m {
		if len(v) <= 0 {
			continue
		}
		if ok := strings.Contains(k, "@"); !ok {
			result[k] = v[0]
			continue
		}
		str := strings.Split(k, "@")
		key, value := str[1], str[0]
		v1, ok := result[key]
		if !ok {
			var l []map[string]interface{}
			for _, t := range v {
				l = append(l, map[string]interface{}{
					value: t,
				})
			}
			result[key] = l
			continue
		}
		v2 := v1.([]map[string]interface{})
		for i, t := range v {
			v2[i][value] = t
		}
		result[key] = v2
	}
	return result
}

func (g *Generator) Unmarshal(s *xlsx.Sheet) ([]byte, error) {
	title, err := s.Row(0)
	if err != nil {
		return nil, err
	}
	var result []map[string]interface{}
	for i := 1; i < s.MaxRow; i++ {
		row, err := s.Row(i)
		if err != nil {
			return nil, err
		}
		result = append(result, g.unmarshal(title, row))
	}
	b, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (g *Generator) WriteTo(s *xlsx.Sheet, w io.Writer) error {
	title, err := s.Row(0)
	if err != nil {
		return err
	}
	titleStr := make(map[string]struct{})
	title.ForEachCell(func(c *xlsx.Cell) error {
		if strings.ToLower(c.Value) == "const" {
			return nil
		}
		titleStr[c.Value] = struct{}{}
		return nil
	})
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("//source %v\n", s.Name))
	builder.WriteString(fmt.Sprintf("package conf\n"))
	builder.WriteString(fmt.Sprintf("type %vRow struct{\n", s.Name))
	arryMap := make(map[string]string)
	for v, _ := range titleStr {
		if ok := strings.Contains(v, "@"); ok {
			str := strings.Split(v, "@")
			arryMap[str[1]] += g.Convert(str[0])
			continue
		}
		builder.WriteString(g.Convert(v))
	}
	for k, v := range arryMap {
		builder.WriteString(fmt.Sprintf("%v []struct{\n", k))
		builder.WriteString(fmt.Sprintf("%v\n", v))
		builder.WriteString(fmt.Sprintf("}\n"))
	}
	builder.WriteString(fmt.Sprintf("}\n"))
	builder.WriteString(fmt.Sprintf("type %vTable []*%vRow\n", s.Name, s.Name))
	builder.WriteString(fmt.Sprintf("var %vConf %vTable\n", s.Name, s.Name))
	for _, v := range g.CodeName(s) {
		if ok := IsAllEnglish(v.Name); !ok {
			continue
		}
		builder.WriteString(fmt.Sprintf("const %v = %v\n", v.Name, v.Code))
	}
	for k, _ := range titleStr {
		switch {
		case strings.Contains(k, "_IDX"):
			builder.WriteString(fmt.Sprintf("func (t %vTable) Get%v(idx int32) (result []*%vRow) {\n", s.Name, k, s.Name))
			builder.WriteString(fmt.Sprintf("for i := 0; i < len(t); i++ {\n"))
			builder.WriteString(fmt.Sprintf("if t[i].%v != idx {\n", k))
			builder.WriteString(fmt.Sprintf("continue"))
			builder.WriteString(fmt.Sprintf("}\n"))
			builder.WriteString("result = append(result, t[i])\n")
			builder.WriteString(fmt.Sprintf("}\n"))
			builder.WriteString(fmt.Sprintf("return result\n"))
			builder.WriteString(fmt.Sprintf("}\n"))
		case strings.Contains(k, "ID_"):
			builder.WriteString(fmt.Sprintf("func (t %vTable) Get%v(idx int32) *%vRow {\n", s.Name, k, s.Name))
			builder.WriteString(fmt.Sprintf("for i := 0; i < len(t); i++ {\n"))
			builder.WriteString(fmt.Sprintf("if t[i].%v != idx {\n", k))
			builder.WriteString(fmt.Sprintf("continue"))
			builder.WriteString(fmt.Sprintf("}\n"))
			builder.WriteString("return t[i]\n")
			builder.WriteString(fmt.Sprintf("}\n"))
			builder.WriteString(fmt.Sprintf("return nil\n"))
			builder.WriteString(fmt.Sprintf("}\n"))
		default:
			continue
		}
	}
	builder.WriteString(fmt.Sprintf("func (t *%vTable) load() {\n", s.Name))
	url := fmt.Sprintf("http://%v/api/conf/%v", "127.0.0.1:8001", strings.ToLower(s.Name))
	builder.WriteString(fmt.Sprintf(`response, err := http.Get("%v")`, url))
	builder.WriteString(fmt.Sprintf("\n"))
	builder.WriteString("defer response.Body.Close()\n")
	builder.WriteString(fmt.Sprintf("if err != nil {\n"))
	builder.WriteString(fmt.Sprintf("panic(err)\n"))
	builder.WriteString(fmt.Sprintf("return\n"))
	builder.WriteString(fmt.Sprintf("}\n"))
	builder.WriteString(fmt.Sprintf("b, err := ioutil.ReadAll(response.Body)\n"))
	builder.WriteString(fmt.Sprintf("if err != nil {\n"))
	builder.WriteString(fmt.Sprintf("panic(err)\n"))
	builder.WriteString(fmt.Sprintf("return\n"))
	builder.WriteString(fmt.Sprintf("}\n"))
	builder.WriteString(fmt.Sprintf("var tb %vTable\n", s.Name))
	builder.WriteString(fmt.Sprintf("if err := json.Unmarshal(b, &tb); err != nil {\n"))
	builder.WriteString(fmt.Sprintf("panic(err)\n"))
	builder.WriteString(fmt.Sprintf("}\n"))
	builder.WriteString(fmt.Sprintf("*t = tb\n"))
	builder.WriteString(fmt.Sprintf("}\n"))
	var t template.Template
	t.Parse(builder.String())
	if err := t.Execute(w, nil); err != nil {
		return err
	}
	return nil
}

func (g *Generator) Gen() error {
	var builder strings.Builder
	builder.WriteString("package conf\n")
	builder.WriteString("func Download() {\n")
	for _, v := range g.Sheets {
		file, err := os.Create(fmt.Sprintf("./conf/%v.go", v.Name))
		if err != nil {
			continue
		}
		if err := g.WriteTo(v, file); err != nil {
			continue
		}
		builder.WriteString(fmt.Sprintf("%vConf.load()\n", v.Name))
	}
	builder.WriteString("}\n")
	var t template.Template
	t.Parse(builder.String())
	loadFile, err := os.Create(fmt.Sprintf("./conf/download.go"))
	if err != nil {
		return err
	}
	if err := t.Execute(loadFile, nil); err != nil {
		return err
	}
	cmd := exec.Command("/bin/bash", "-c", fmt.Sprintf("gofmt -w conf"))
	if err := cmd.Run(); err != nil {
		panic(err)
	}
	cmd = exec.Command("/bin/bash", "-c", fmt.Sprintf("goimports -w conf"))
	if err := cmd.Run(); err != nil {
		panic(err)
	}
	return nil
}

func (g *Generator) CodeName(s *xlsx.Sheet) (result []*CodeName) {
	row, err := s.Row(0)
	if err != nil {
		return result
	}
	var idIdx, nameIdx int
	for i := 0; i < row.Sheet.MaxCol; i++ {
		c := row.GetCell(i)
		if strings.ToLower(c.Value) == "const" {
			nameIdx = i
		}
		if strings.ToLower(c.Value) == "id_int" {
			idIdx = i
		}
	}
	if nameIdx <= 0 && idIdx <= 0 {
		return nil
	}
	for i := 1; i < s.MaxRow; i++ {
		r, err := s.Row(i)
		if err != nil {
			return result
		}
		number, err := strconv.Atoi(r.GetCell(idIdx).Value)
		if err != nil {
			return result
		}
		result = append(result, &CodeName{
			Name: r.GetCell(nameIdx).Value,
			Code: int32(number),
		})
	}
	return result
}

func IsAllEnglish(str string) bool {
	for _, v := range str {
		if v >= 'A' && v <= 'Z' {
			continue
		}
		if v >= 'a' && v <= 'z' {
			continue
		}
		return false
	}
	return true
}

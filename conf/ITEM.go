//source ITEM
package conf

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type ITEMRow struct {
	ITEM_NAME_STR       string
	ITEM_TYPE_CONST_IDX int32
	ID_INT              int32
	ICON_STR            string
	BUY                 []struct {
		BUY_ITEM_TYPE_CONST int32
		BUY_ITEM_COUNT_INT  int32
	}
}
type ITEMTable []*ITEMRow

var ITEMConf ITEMTable

func (t ITEMTable) GetITEM_TYPE_CONST_IDX(idx int32) (result []*ITEMRow) {
	for i := 0; i < len(t); i++ {
		if t[i].ITEM_TYPE_CONST_IDX != idx {
			continue
		}
		result = append(result, t[i])
	}
	return result
}
func (t ITEMTable) GetID_INT(idx int32) *ITEMRow {
	for i := 0; i < len(t); i++ {
		if t[i].ID_INT != idx {
			continue
		}
		return t[i]
	}
	return nil
}
func (t *ITEMTable) load() {
	response, err := http.Get("http://127.0.0.1:8001/api/conf/item")
	defer response.Body.Close()
	if err != nil {
		panic(err)
		return
	}
	b, err := ioutil.ReadAll(response.Body)
	if err != nil {
		panic(err)
		return
	}
	var tb ITEMTable
	if err := json.Unmarshal(b, &tb); err != nil {
		panic(err)
	}
	*t = tb
}

//source ITEM_TYPE
package conf

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type ITEM_TYPERow struct {
	ID_INT int32
}
type ITEM_TYPETable []*ITEM_TYPERow

var ITEM_TYPEConf ITEM_TYPETable

const SWORDS = 1
const CLOTHES = 2
const SHOES = 3
const RES = 4

func (t ITEM_TYPETable) GetID_INT(idx int32) *ITEM_TYPERow {
	for i := 0; i < len(t); i++ {
		if t[i].ID_INT != idx {
			continue
		}
		return t[i]
	}
	return nil
}
func (t *ITEM_TYPETable) load() {
	response, err := http.Get("http://127.0.0.1:8001/api/conf/item_type")
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
	var tb ITEM_TYPETable
	if err := json.Unmarshal(b, &tb); err != nil {
		panic(err)
	}
	*t = tb
}

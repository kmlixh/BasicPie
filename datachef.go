package basicPie

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/kmlixh/gom/v2"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

type BeforeSave func(c *gin.Context, i interface{}) (bool, interface{}, error)

type DataChef struct {
	Type       reflect.Type
	TableModel gom.TableModel
	Db         *gom.DB
	BeforeSave
}

func GenerateChef(i interface{}, db *gom.DB, save BeforeSave) *DataChef {
	typ := reflect.TypeOf(i)
	model, er := gom.GetTableModel(i)
	if er != nil {
		panic(er)
	}
	return &DataChef{typ, model, db, save}
}
func GetCondtionMapFromRst(c *gin.Context) (map[string]interface{}, error) {
	var maps map[string]interface{}
	var er error
	if c.Request.Method == "POST" {
		contentType := c.GetHeader("Content-Type")
		if strings.Contains(contentType, "application/x-www-form-urlencoded") {
			maps = make(map[string]interface{})
			er = c.Request.ParseForm()
			if er != nil {
				return nil, er
			}
			values := c.Request.Form
			for k, v := range values {
				if len(v) == 1 {
					maps[k] = v[0]
				} else {
					maps[k] = v
				}
			}

		} else if strings.Contains(contentType, "application/json") {
			bbs, er := io.ReadAll(c.Request.Body)
			if er != nil {
				return nil, er
			}
			er = json.Unmarshal(bbs, &maps)
		}
	} else if c.Request.Method == http.MethodGet {
		maps = make(map[string]interface{})
		values := c.Request.URL.Query()
		for k, v := range values {
			if len(v) == 1 {
				maps[k] = v[0]
			} else {
				maps[k] = v
			}
		}
	}
	if er != nil {
		return nil, er
	}
	return maps, nil

}

func (d DataChef) Cook(route gin.IRoutes, name string) {
	route.Any("/"+name+"/query", func(c *gin.Context) {
		var result interface{}
		maps, er := GetCondtionMapFromRst(c)
		if er != nil {
			RenderJson(c, Err2(500, er.Error()))
			return
		}
		cnd := gom.MapToCondition(maps)
		result, er = d.Db.Where(cnd).Select(reflect.New(d.Type).Interface())
		if er != nil {
			RenderJson(c, Err2(500, er.Error()))
		} else {
			RenderJson(c, Ok(result))
		}
	})
	route.Any("/"+name+"/delete", func(c *gin.Context) {
		maps, er := GetCondtionMapFromRst(c)
		if er != nil {
			RenderJson(c, Err2(500, er.Error()))
			return
		}
		key := d.TableModel.Columns()[0]
		val, ok := maps[key]
		if !ok {
			RenderJson(c, Err2(404, "not find key"))
			return
		}
		cnd := gom.MapToCondition(map[string]interface{}{key: val})
		i, _, er := d.Db.Where(cnd).Delete(reflect.New(d.Type).Interface())
		if er != nil {
			RenderJson(c, Err2(500, er.Error()))
		} else {
			RenderJson(c, Ok(i))
		}
	})
	route.Any("/"+name+"/list", func(c *gin.Context) {
		var results interface{}
		if !d.TableModel.PrimaryAuto() {
			RenderJson(c, Err())
		}
		results = reflect.New(reflect.SliceOf(d.Type)).Interface()
		page := int64(0)
		pageSize := int64(0)
		maps, er := GetCondtionMapFromRst(c)
		if er != nil {
			RenderJson(c, Err2(500, er.Error()))
			return
		}
		orderByKey, ook := d.TableModel.Columns()[0], true
		orderByData, odk := maps["id"]
		mode, otk := maps["mode"]
		pSize, okp := maps["pageSize"]
		if !okp {
			pSize = "20"
		}
		if okp {
			delete(maps, "pageSize")
			switch pSize.(type) {
			case string:
				t, er := strconv.Atoi(pSize.(string))
				if er == nil {
					pageSize = int64(t)
				}
			case float32:
				pageSize = int64(pSize.(float32))
			case float64:
				pageSize = int64(pSize.(float64))
			case int32:
				pageSize = int64(pSize.(int32))
			case int:
				pageSize = int64(pSize.(int))
			case int16:
				pageSize = int64(pSize.(int16))
			case int8:
				pageSize = int64(pSize.(int8))
			}
		}

		if er != nil {
			pageSize = 20
		}

		if ook && otk {
			//以排序值滚动获取数据
			delete(maps, "id")
			delete(maps, "mode")

			cnd := gom.MapToCondition(maps)
			orderType := gom.Desc
			if odk && mode == "1" {
				orderType = gom.Asc
				cnd.Gt(orderByKey, orderByData)
			}
			if odk && mode == "0" {
				cnd.Lt(orderByKey, orderByData)
			}
			d.Db.Where(cnd).OrderBy(orderByKey, orderType).Page(0, int64(pageSize)).Select(results)
			RenderJson(c, Ok(results).Set("pageSize", pageSize))
		} else {
			pTxt, ok := maps["page"]
			if !ok {
				page = 1
			} else {
				delete(maps, "page")
				switch pTxt.(type) {
				case string:
					t, er := strconv.Atoi(pTxt.(string))
					if er == nil {
						page = int64(t)
					}
				case float32:
					page = int64(pTxt.(float32))
				case float64:
					page = int64(pTxt.(float64))
				case int32:
					page = int64(pTxt.(int32))
				case int:
					page = int64(pTxt.(int))
				case int16:
					page = int64(pTxt.(int16))
				case int8:
					page = int64(pTxt.(int8))
				}

			}

			if er != nil {
				page = 1
			}

			totalPages := int64(0)
			var cnd gom.Condition
			cnd = gom.MapToCondition(maps)

			count, er := d.Db.Where(cnd).Table(d.TableModel.Table()).Count(d.TableModel.Columns()[0])
			z := int64(0)
			if count%int64(pageSize) > 0 {
				z = 1
			}
			totalPages = count/int64(pageSize) + z
			_, er = d.Db.Where(cnd).Page(int64(page), int64(pageSize)).Select(results)
			if er != nil {
				RenderJson(c, Err2(500, er.Error()))
				return
			}
			codeMsg := Ok()
			codeMsg["data"] = map[string]interface{}{"list": results, "page": page, "pageSize": pageSize, "totalPages": totalPages}
			RenderJson(c, codeMsg)
		}

	})
	route.POST("/"+name+"/save", func(c *gin.Context) {
		var i interface{}
		var er error
		id := int64(0)
		isInsert := false
		i, er = d.defaultJsonParser(c)
		if er != nil {
			RenderJson(c, Err2(500, er.Error()))
			return
		}
		if d.BeforeSave == nil {
			RenderJson(c, Err2(500, "please add a [BeforeSaveFunc]"))
			return
		}
		isInsert, i, er = d.BeforeSave(c, i)
		if er != nil {
			RenderJson(c, Err2(500, er.Error()))
			return
		}
		if isInsert {
			_, id, er = d.Db.Insert(i)
		} else {
			_, _, er = d.Db.Update(i)
		}
		if er != nil {
			RenderJson(c, Err2(500, er.Error()))
			return
		}
		RenderJson(c, Ok(id))
	})
}
func (d DataChef) defaultJsonParser(c *gin.Context) (interface{}, error) {
	i := reflect.New(d.Type).Interface()
	if c.Request.Method == "POST" {
		contentType := c.GetHeader("Content-Type")
		if strings.Contains(contentType, "application/json") {
			bbs, er := io.ReadAll(c.Request.Body)
			if er != nil {
				return nil, er
			}
			json.Unmarshal(bbs, i)
		}
	}
	return i, nil
}

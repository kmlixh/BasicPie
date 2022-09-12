package BasicPie

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

func generateChef(typ reflect.Type, model gom.TableModel, db *gom.DB, save BeforeSave) *DataChef {
	return &DataChef{typ, model, db, save}
}
func GetCondtionMapFromRst(c *gin.Context) (map[string]interface{}, error) {
	maps := make(map[string]interface{})
	var er error
	if c.Request.Method == "POST" {
		contentType := c.GetHeader("Content-Type")
		if strings.Contains(contentType, "application/x-www-form-urlencoded") {
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
			er = json.Unmarshal(bbs, maps)
		}
	} else if c.Request.Method == http.MethodGet {
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
			renderJson(c, Err2(500, er.Error()))
			return
		}
		cnd := gom.MapToCondition(maps)
		result, er = d.Db.Where(cnd).Select(reflect.New(d.Type).Interface())
		if er != nil {
			renderJson(c, Err2(500, er.Error()))
		} else {
			renderJson(c, Ok(result))
		}
	})
	route.Any("/"+name+"/delete", func(c *gin.Context) {
		maps, er := GetCondtionMapFromRst(c)
		if er != nil {
			renderJson(c, Err2(500, er.Error()))
			return
		}
		cnd := gom.MapToCondition(maps)
		i, _, er := d.Db.Where(cnd).Delete(reflect.New(d.Type).Interface())
		if er != nil {
			renderJson(c, Err2(500, er.Error()))
		} else {
			renderJson(c, Ok(i))
		}
	})
	route.Any("/"+name+"/list", func(c *gin.Context) {
		var results interface{}
		if !d.TableModel.PrimaryAuto() {
			renderJson(c, Err())
		}
		results = reflect.New(reflect.SliceOf(d.Type)).Interface()

		maps, er := GetCondtionMapFromRst(c)
		if er != nil {
			renderJson(c, Err2(500, er.Error()))
			return
		}
		orderByKey, ook := maps["oBKey"].(string)
		orderByData, odk := maps["oBData"]
		orderByType, otk := maps["oBType"].(string)
		pSize, okp := maps["pageSize"].(string)
		if okp {
			delete(maps, "pageSize")
		}
		pageSize, er := strconv.Atoi(pSize)
		if er != nil {
			pageSize = 20
		}
		if !okp {
			pSize = "20"
		}
		if ook && otk {
			//以排序值滚动获取数据
			delete(maps, "oBKey")
			delete(maps, "oBData")
			delete(maps, "oBType")

			cnd := gom.MapToCondition(maps)
			orderType := gom.Desc
			if odk && orderByType == "new" {
				orderType = gom.Asc
				cnd.Gt(orderByKey, orderByData)
			}
			if odk && orderByType == "old" {
				cnd.Lt(orderByKey, orderByData)
			}
			d.Db.Where(cnd).OrderBy(orderByKey, orderType).Page(0, int64(pageSize)).Select(results)
			renderJson(c, Ok(results).Set("pageSize", pageSize))
		} else {
			pTxt, ok := maps["page"].(string)
			if !ok {
				pTxt = "0"
			} else {
				delete(maps, "page")
			}

			page, er := strconv.Atoi(pTxt)
			if er != nil {
				page = 0
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
				renderJson(c, Err2(500, er.Error()))
				return
			}
			codeMsg := Ok()
			codeMsg["data"] = map[string]interface{}{"list": results, "page": page, "pageSize": pageSize, "totalPages": totalPages}
			renderJson(c, codeMsg)
		}

	})
	route.POST("/"+name+"/save", func(c *gin.Context) {
		var i interface{}
		var er error
		id := int64(0)
		isInsert := false
		i, er = d.defaultJsonParser(c)
		if er != nil {
			renderJson(c, Err2(500, er.Error()))
			return
		}
		if d.BeforeSave == nil {
			renderJson(c, Err2(500, "please add a [BeforeSaveFunc]"))
			return
		}
		isInsert, i, er = d.BeforeSave(c, i)
		if er != nil {
			renderJson(c, Err2(500, er.Error()))
			return
		}
		if isInsert {
			_, id, er = d.Db.Insert(i)
		} else {
			_, _, er = d.Db.Update(i)
		}
		if er != nil {
			renderJson(c, Err2(500, er.Error()))
			return
		}
		renderJson(c, Ok(id))
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

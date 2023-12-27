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

type FilterFunc func(chain FilterChain, handler AutoRequestHandler, context *gin.Context) (bool, error, interface{})

type FilterChain struct {
	filterMap map[CrudType][]Filter
}

func (f FilterChain) runFilters(handler AutoRequestHandler, crudType CrudType, context *gin.Context) (bool, error, interface{}) {
	filters := f.filterMap[crudType]
	if filters == nil || len(filters) == 0 {
		return true, nil, nil
	}
	for idx, filter := range filters {
		b, e, i := filter.filterFunc(f, handler, context)
		if !b || idx == len(filters)-1 {
			return b, e, i
		}
	}
	return true, nil, nil
}

func filtersToFilterChain(filters []Filter) *FilterChain {
	maps := make(map[CrudType][]Filter)
	for _, f := range filters {
		groupFilters := maps[f.CrudType]
		if groupFilters == nil {
			groupFilters = make([]Filter, 0)
			groupFilters = append(groupFilters, f)
			maps[f.CrudType] = groupFilters
		}
	}
	return &FilterChain{maps}
}

type Filter struct {
	CrudType
	filterFunc FilterFunc
}

type QueryOperator int

const (
	_ QueryOperator = iota
	Le
	Lt
	Ge
	Gt
	Eq
	Like
	LikeLeft
	LikeRight
)

type CrudType int

const (
	_ CrudType = iota
	BeforeQuery
	DoQuery
	AfterQuery
	BeforeQuerySingle
	DoQuerySingle
	AfterQuerySingle
	BeforeUpdate
	DoUpdate
	AfterUpdate
	BeforeInsert
	DoInsert
	AfterInsert
	BeforeDelete
	DoDelete
	AfterDelete
)

type QueryCnd struct {
	QueryName string
	InnerName string
	QueryOperator
}

type AutoRequestHandler struct {
	Db        *gom.DB
	FilterMap map[CrudType]Filter
	QueryCnds []QueryCnd
}

func CreateHandler(i interface{}, db *gom.DB, filters []Filter) *AutoRequestHandler {

	return &AutoRequestHandler{db, nil, nil}
}
func GetConditionMapFromRst(c *gin.Context) (map[string]interface{}, error) {
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

func (d AutoRequestHandler) getConditionFromMap(maps map[string]interface{}) gom.Condition {

}

func (d AutoRequestHandler) Handle(route gin.IRoutes, name string) {
	route.Any("/"+name+"/query", func(c *gin.Context) {
		if d.CustomDetail != nil {
			d.CustomDetail(c)
			return
		} else {
			var result interface{}
			maps, er := GetConditionMapFromRst(c)
			if er != nil {
				RenderJson(c, Err2(500, er.Error()))
				return
			}
			cnd := d.getConditionFromMap(maps)
			result, er = d.Db.Where(cnd).Select(reflect.New(d.Type).Interface())
			if er != nil {
				RenderJson(c, Err2(500, er.Error()))
			} else {
				RenderJson(c, Ok(result))
			}
		}
	})
	route.Any("/"+name+"/delete", func(c *gin.Context) {
		maps, er := GetConditionMapFromRst(c)
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
			return
		}
		results = reflect.New(reflect.SliceOf(d.Type)).Interface()
		page := int64(0)
		pageSize := int64(20)
		maps, er := GetConditionMapFromRst(c)
		if er != nil {
			RenderJson(c, Err2(500, er.Error()))
			return
		}
		orderByKey := d.TableModel.Columns()[0]
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
		if pageSize == 0 {
			pageSize = 20
		}
		var cols []string
		if len(d.Columns) > 0 {
			cols = d.Columns
		}
		if otk {
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

			d.Db.Where(cnd).OrderBy(orderByKey, orderType).Page(0, pageSize).Select(results, cols...)
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
			var orderBys []gom.OrderBy
			if d.OrderBys != nil && len(d.OrderBys) > 0 {
				orderBys = append(orderBys, d.OrderBys...)

			} else {
				orderBys = []gom.OrderBy{gom.MakeOrderBy(orderByKey, gom.Desc)}
			}
			_, er = d.Db.Where(cnd).Page(int64(page), int64(pageSize)).OrderBys(orderBys).Select(results, cols...)
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
		if d.CustomSave != nil {
			result := d.CustomSave(c)
			if result {
				RenderJson(c, Ok())
			} else {
				RenderJson(c, Err())
			}
			return
		}
		id := int64(0)
		isInsert := false
		i := reflect.New(d.Type).Interface()
		er := c.ShouldBind(i)
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

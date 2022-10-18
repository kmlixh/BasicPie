package basicPie

import "github.com/gin-gonic/gin"

type CodeMsg map[string]interface{}

func Ok(data ...interface{}) CodeMsg {
	c := RawCodeMsg(0, "ok", nil)
	if data != nil && len(data) > 0 {
		c.SetData(data[0])
	}
	return c
}

func (c CodeMsg) Code() int {
	return c["code"].(int)
}
func (c CodeMsg) SetCode(code int) CodeMsg {
	c["code"] = code
	return c
}
func (c CodeMsg) Msg() string {
	return c["msg"].(string)
}
func (c CodeMsg) SetMsg(msg string) CodeMsg {
	c["msg"] = msg
	return c
}
func (c CodeMsg) Data() int {
	return c["data"].(int)
}
func (c CodeMsg) SetData(data interface{}) CodeMsg {
	c.Set("data", data)
	return c
}
func (c CodeMsg) Set(name string, data interface{}) CodeMsg {
	c[name] = data
	return c
}
func RawCodeMsg(code int, msg string, data interface{}) CodeMsg {
	codeMsg := CodeMsg{}
	codeMsg["code"] = code
	codeMsg["msg"] = msg
	if data != nil {
		codeMsg["data"] = data
	}
	return codeMsg
}
func Err() CodeMsg {
	return RawCodeMsg(-1, "error", nil)
}

func Err2(code int, msg string) CodeMsg {
	return RawCodeMsg(code, msg, nil)
}
func RenderJson(c *gin.Context, data interface{}) {
	c.JSON(200, data)
}
func RenderOk(c *gin.Context, data ...interface{}) {
	c.JSON(200, Ok(data...))
}
func RenderErr(c *gin.Context) {
	c.JSON(200, Err())
}
func RenderErr2(c *gin.Context, code int, msg string) {
	c.JSON(200, Err2(code, msg))
}

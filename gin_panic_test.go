package gin

import (
	"fmt"
	"net/http"
	"testing"
)

func TestPanic(t *testing.T) {
	r := New()
	r.Use(Logger())

	r.Use(CustomRecovery(func(c *Context, err interface{}) {
		if err != nil{
			c.AbortWithStatusJSON(http.StatusBadGateway,fmt.Sprint("[出错了][CustomRecovery]err=====",err))
		}
	}))

	r.GET("/panic", func(context *Context) {
		var v map[string]interface{}
		v["1"] = 2
		context.String(http.StatusOK,"panic")
	})
	r.Run(":80")
}


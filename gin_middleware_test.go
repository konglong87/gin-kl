package gin

import (
	"fmt"
	"testing"
)

var mw1 = func(ctx *Context) {
	fmt.Println("[before] this is mw1 .")
	ctx.Next()
	fmt.Println("[end] this is mw1 .")
}
var mw2 = func(ctx *Context) {
	fmt.Println("[before] this is mw2 .")
	ctx.Next()
	fmt.Println("[end] this is mw2 .")
}
var mw3 = func(ctx *Context) {
	fmt.Println("[before] this is mw3 .")
	fmt.Println("[ing] this is mw3 .")
	ctx.Next()
}

var mw4 = func(ctx *Context) {
	ctx.Next()
	fmt.Println("[after next] this is mw4 .")
	fmt.Println("[after next] this is mw4 .")
}

var mw5 = func(ctx *Context) {
	fmt.Println("[before next] this is mw5 .")
	ctx.Next()
	fmt.Println("[after next] this is mw5 .")
}
func TestMiddlewares(t *testing.T){
	e1 := New()
	e1.Use(mw1,mw2,mw3,mw4,mw5)

	e1.Group("/v1")
	e1.GET("/name", func(context *Context) {
		fmt.Println("getName")
	})
	e1.Run(":80")
}


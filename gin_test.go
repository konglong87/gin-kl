// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package gin

import (
	"crypto/tls"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func formatAsDate(t time.Time) string {
	year, month, day := t.Date()
	return fmt.Sprintf("%d/%02d/%02d", year, month, day)
}

func setupHTMLFiles(t *testing.T, mode string, tls bool, loadMethod func(*Engine)) *httptest.Server {
	SetMode(mode)
	defer SetMode(TestMode)

	var router *Engine
	captureOutput(t, func() {
		router = New()
		router.Delims("{[{", "}]}")
		router.SetFuncMap(template.FuncMap{
			"formatAsDate": formatAsDate,
		})
		loadMethod(router)
		router.GET("/test", func(c *Context) {
			c.HTML(http.StatusOK, "hello.tmpl", map[string]string{"name": "world"})
		})
		router.GET("/raw", func(c *Context) {
			c.HTML(http.StatusOK, "raw.tmpl", map[string]interface{}{
				"now": time.Date(2017, 07, 01, 0, 0, 0, 0, time.UTC),
			})
		})
	})

	var ts *httptest.Server

	if tls {
		ts = httptest.NewTLSServer(router)
	} else {
		ts = httptest.NewServer(router)
	}

	return ts
}

func TestLoadHTMLGlobDebugMode(t *testing.T) {
	ts := setupHTMLFiles(
		t,
		DebugMode,
		false,
		func(router *Engine) {
			router.LoadHTMLGlob("./testdata/template/*")
		},
	)
	defer ts.Close()

	res, err := http.Get(fmt.Sprintf("%s/test", ts.URL))
	if err != nil {
		fmt.Println(err)
	}

	resp, _ := ioutil.ReadAll(res.Body)
	assert.Equal(t, "<h1>Hello world</h1>", string(resp))
}

func TestLoadHTMLGlobTestMode(t *testing.T) {
	ts := setupHTMLFiles(
		t,
		TestMode,
		false,
		func(router *Engine) {
			router.LoadHTMLGlob("./testdata/template/*")
		},
	)
	defer ts.Close()

	res, err := http.Get(fmt.Sprintf("%s/test", ts.URL))
	if err != nil {
		fmt.Println(err)
	}

	resp, _ := ioutil.ReadAll(res.Body)
	assert.Equal(t, "<h1>Hello world</h1>", string(resp))
}

func TestLoadHTMLGlobReleaseMode(t *testing.T) {
	ts := setupHTMLFiles(
		t,
		ReleaseMode,
		false,
		func(router *Engine) {
			router.LoadHTMLGlob("./testdata/template/*")
		},
	)
	defer ts.Close()

	res, err := http.Get(fmt.Sprintf("%s/test", ts.URL))
	if err != nil {
		fmt.Println(err)
	}

	resp, _ := ioutil.ReadAll(res.Body)
	assert.Equal(t, "<h1>Hello world</h1>", string(resp))
}

func TestLoadHTMLGlobUsingTLS(t *testing.T) {
	ts := setupHTMLFiles(
		t,
		DebugMode,
		true,
		func(router *Engine) {
			router.LoadHTMLGlob("./testdata/template/*")
		},
	)
	defer ts.Close()

	// Use InsecureSkipVerify for avoiding `x509: certificate signed by unknown authority` error
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{Transport: tr}
	res, err := client.Get(fmt.Sprintf("%s/test", ts.URL))
	if err != nil {
		fmt.Println(err)
	}

	resp, _ := ioutil.ReadAll(res.Body)
	assert.Equal(t, "<h1>Hello world</h1>", string(resp))
}

func TestLoadHTMLGlobFromFuncMap(t *testing.T) {
	ts := setupHTMLFiles(
		t,
		DebugMode,
		false,
		func(router *Engine) {
			router.LoadHTMLGlob("./testdata/template/*")
		},
	)
	defer ts.Close()

	res, err := http.Get(fmt.Sprintf("%s/raw", ts.URL))
	if err != nil {
		fmt.Println(err)
	}

	resp, _ := ioutil.ReadAll(res.Body)
	assert.Equal(t, "Date: 2017/07/01\n", string(resp))
}

//设置 test mode，默认 debug
func init() {
	SetMode(TestMode)
}

func TestCreateEngine(t *testing.T) {
	router := New()
	assert.Equal(t, "/", router.basePath)
	assert.Equal(t, router.engine, router)
	assert.Empty(t, router.Handlers)
}

func TestLoadHTMLFilesTestMode(t *testing.T) {
	ts := setupHTMLFiles(
		t,
		TestMode,
		false,
		func(router *Engine) {
			router.LoadHTMLFiles("./testdata/template/hello.tmpl", "./testdata/template/raw.tmpl")
		},
	)
	defer ts.Close()

	res, err := http.Get(fmt.Sprintf("%s/test", ts.URL))
	if err != nil {
		fmt.Println(err)
	}

	resp, _ := ioutil.ReadAll(res.Body)
	assert.Equal(t, "<h1>Hello world</h1>", string(resp))
}

func TestLoadHTMLFilesDebugMode(t *testing.T) {
	ts := setupHTMLFiles(
		t,
		DebugMode,
		false,
		func(router *Engine) {
			router.LoadHTMLFiles("./testdata/template/hello.tmpl", "./testdata/template/raw.tmpl")
		},
	)
	defer ts.Close()

	res, err := http.Get(fmt.Sprintf("%s/test", ts.URL))
	if err != nil {
		fmt.Println(err)
	}

	resp, _ := ioutil.ReadAll(res.Body)
	assert.Equal(t, "<h1>Hello world</h1>", string(resp))
}

func TestLoadHTMLFilesReleaseMode(t *testing.T) {
	ts := setupHTMLFiles(
		t,
		ReleaseMode,
		false,
		func(router *Engine) {
			router.LoadHTMLFiles("./testdata/template/hello.tmpl", "./testdata/template/raw.tmpl")
		},
	)
	defer ts.Close()

	res, err := http.Get(fmt.Sprintf("%s/test", ts.URL))
	if err != nil {
		fmt.Println(err)
	}

	resp, _ := ioutil.ReadAll(res.Body)
	assert.Equal(t, "<h1>Hello world</h1>", string(resp))
}

func TestLoadHTMLFilesUsingTLS(t *testing.T) {
	ts := setupHTMLFiles(
		t,
		TestMode,
		true,
		func(router *Engine) {
			router.LoadHTMLFiles("./testdata/template/hello.tmpl", "./testdata/template/raw.tmpl")
		},
	)
	defer ts.Close()

	// Use InsecureSkipVerify for avoiding `x509: certificate signed by unknown authority` error
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{Transport: tr}
	res, err := client.Get(fmt.Sprintf("%s/test", ts.URL))
	if err != nil {
		fmt.Println(err)
	}

	resp, _ := ioutil.ReadAll(res.Body)
	assert.Equal(t, "<h1>Hello world</h1>", string(resp))
}

func TestLoadHTMLFilesFuncMap(t *testing.T) {
	ts := setupHTMLFiles(
		t,
		TestMode,
		false,
		func(router *Engine) {
			router.LoadHTMLFiles("./testdata/template/hello.tmpl", "./testdata/template/raw.tmpl")
		},
	)
	defer ts.Close()

	res, err := http.Get(fmt.Sprintf("%s/raw", ts.URL))
	if err != nil {
		fmt.Println(err)
	}

	resp, _ := ioutil.ReadAll(res.Body)
	assert.Equal(t, "Date: 2017/07/01\n", string(resp))
}

func TestAddRoute(t *testing.T) {
	router := New()
	router.addRoute("GET", "/", HandlersChain{func(_ *Context) {}})

	assert.Len(t, router.trees, 1)
	assert.NotNil(t, router.trees.get("GET"))
	assert.Nil(t, router.trees.get("POST"))

	router.addRoute("POST", "/", HandlersChain{func(_ *Context) {}})

	assert.Len(t, router.trees, 2)
	assert.NotNil(t, router.trees.get("GET"))
	assert.NotNil(t, router.trees.get("POST"))

	router.addRoute("POST", "/post", HandlersChain{func(_ *Context) {}})
	assert.Len(t, router.trees, 2)
}

func TestAddRouteFails(t *testing.T) {
	router := New()
	assert.Panics(t, func() { router.addRoute("", "/", HandlersChain{func(_ *Context) {}}) })
	assert.Panics(t, func() { router.addRoute("GET", "a", HandlersChain{func(_ *Context) {}}) })
	assert.Panics(t, func() { router.addRoute("GET", "/", HandlersChain{}) })

	router.addRoute("POST", "/post", HandlersChain{func(_ *Context) {}})
	assert.Panics(t, func() {
		router.addRoute("POST", "/post", HandlersChain{func(_ *Context) {}})
	})
}

func TestCreateDefaultRouter(t *testing.T) {
	router := Default()
	assert.Len(t, router.Handlers, 2)
}

func TestNoRouteWithoutGlobalHandlers(t *testing.T) {
	var middleware0 HandlerFunc = func(c *Context) {}
	var middleware1 HandlerFunc = func(c *Context) {}

	router := New()

	router.NoRoute(middleware0)
	assert.Nil(t, router.Handlers)
	assert.Len(t, router.noRoute, 1)
	assert.Len(t, router.allNoRoute, 1)
	compareFunc(t, router.noRoute[0], middleware0)
	compareFunc(t, router.allNoRoute[0], middleware0)

	router.NoRoute(middleware1, middleware0)
	assert.Len(t, router.noRoute, 2)
	assert.Len(t, router.allNoRoute, 2)
	compareFunc(t, router.noRoute[0], middleware1)
	compareFunc(t, router.allNoRoute[0], middleware1)
	compareFunc(t, router.noRoute[1], middleware0)
	compareFunc(t, router.allNoRoute[1], middleware0)
}

func TestNoRouteWithGlobalHandlers(t *testing.T) {
	var middleware0 HandlerFunc = func(c *Context) {}
	var middleware1 HandlerFunc = func(c *Context) {}
	var middleware2 HandlerFunc = func(c *Context) {}

	router := New()
	router.Use(middleware2)

	router.NoRoute(middleware0)
	assert.Len(t, router.allNoRoute, 2)
	assert.Len(t, router.Handlers, 1)
	assert.Len(t, router.noRoute, 1)

	compareFunc(t, router.Handlers[0], middleware2)
	compareFunc(t, router.noRoute[0], middleware0)
	compareFunc(t, router.allNoRoute[0], middleware2)
	compareFunc(t, router.allNoRoute[1], middleware0)

	router.Use(middleware1)
	assert.Len(t, router.allNoRoute, 3)
	assert.Len(t, router.Handlers, 2)
	assert.Len(t, router.noRoute, 1)

	compareFunc(t, router.Handlers[0], middleware2)
	compareFunc(t, router.Handlers[1], middleware1)
	compareFunc(t, router.noRoute[0], middleware0)
	compareFunc(t, router.allNoRoute[0], middleware2)
	compareFunc(t, router.allNoRoute[1], middleware1)
	compareFunc(t, router.allNoRoute[2], middleware0)
}

func TestNoMethodWithoutGlobalHandlers(t *testing.T) {
	var middleware0 HandlerFunc = func(c *Context) {}
	var middleware1 HandlerFunc = func(c *Context) {}

	router := New()

	router.NoMethod(middleware0)
	assert.Empty(t, router.Handlers)
	assert.Len(t, router.noMethod, 1)
	assert.Len(t, router.allNoMethod, 1)
	compareFunc(t, router.noMethod[0], middleware0)
	compareFunc(t, router.allNoMethod[0], middleware0)

	router.NoMethod(middleware1, middleware0)
	assert.Len(t, router.noMethod, 2)
	assert.Len(t, router.allNoMethod, 2)
	compareFunc(t, router.noMethod[0], middleware1)
	compareFunc(t, router.allNoMethod[0], middleware1)
	compareFunc(t, router.noMethod[1], middleware0)
	compareFunc(t, router.allNoMethod[1], middleware0)
}

func TestRebuild404Handlers(t *testing.T) {

}

func TestNoMethodWithGlobalHandlers(t *testing.T) {
	var middleware0 HandlerFunc = func(c *Context) {}
	var middleware1 HandlerFunc = func(c *Context) {}
	var middleware2 HandlerFunc = func(c *Context) {}

	router := New()
	router.Use(middleware2)

	router.NoMethod(middleware0)
	assert.Len(t, router.allNoMethod, 2)
	assert.Len(t, router.Handlers, 1)
	assert.Len(t, router.noMethod, 1)

	compareFunc(t, router.Handlers[0], middleware2)
	compareFunc(t, router.noMethod[0], middleware0)
	compareFunc(t, router.allNoMethod[0], middleware2)
	compareFunc(t, router.allNoMethod[1], middleware0)

	router.Use(middleware1)
	assert.Len(t, router.allNoMethod, 3)
	assert.Len(t, router.Handlers, 2)
	assert.Len(t, router.noMethod, 1)

	compareFunc(t, router.Handlers[0], middleware2)
	compareFunc(t, router.Handlers[1], middleware1)
	compareFunc(t, router.noMethod[0], middleware0)
	compareFunc(t, router.allNoMethod[0], middleware2)
	compareFunc(t, router.allNoMethod[1], middleware1)
	compareFunc(t, router.allNoMethod[2], middleware0)
}

func compareFunc(t *testing.T, a, b interface{}) {
	sf1 := reflect.ValueOf(a)
	sf2 := reflect.ValueOf(b)
	if sf1.Pointer() != sf2.Pointer() {
		t.Error("different functions")
	}
}

func TestListOfRoutes(t *testing.T) {
	router := New()
	//router.Handle()
	router.GET("/favicon.ico", handlerTest1)
	router.GET("/", handlerTest1)
	group := router.Group("/users")
	{
		group.GET("/", handlerTest2)
		group.GET("/:id", handlerTest1)
		group.POST("/:id", handlerTest2)
	}
	router.Static("/static", ".")

	list := router.Routes()

	assert.Len(t, list, 7)
	assertRoutePresent(t, list, RouteInfo{
		Method:  "GET",
		Path:    "/favicon.ico",
		Handler: "^(.*/vendor/)?github.com/gin-gonic/gin.handlerTest1$",
	})
	assertRoutePresent(t, list, RouteInfo{
		Method:  "GET",
		Path:    "/",
		Handler: "^(.*/vendor/)?github.com/gin-gonic/gin.handlerTest1$",
	})
	assertRoutePresent(t, list, RouteInfo{
		Method:  "GET",
		Path:    "/users/",
		Handler: "^(.*/vendor/)?github.com/gin-gonic/gin.handlerTest2$",
	})
	assertRoutePresent(t, list, RouteInfo{
		Method:  "GET",
		Path:    "/users/:id",
		Handler: "^(.*/vendor/)?github.com/gin-gonic/gin.handlerTest1$",
	})
	assertRoutePresent(t, list, RouteInfo{
		Method:  "POST",
		Path:    "/users/:id",
		Handler: "^(.*/vendor/)?github.com/gin-gonic/gin.handlerTest2$",
	})
}

func TestEngineHandleContext(t *testing.T) {
	r := New()
	r.GET("/", func(c *Context) {
		c.Request.URL.Path = "/v2"
		r.HandleContext(c)
	})
	v2 := r.Group("/v2")
	{
		v2.GET("/", func(c *Context) {})
	}

	assert.NotPanics(t, func() {
		w := performRequest(r, "GET", "/")
		assert.Equal(t, 301, w.Code)
	})
}

func TestEngineHandleContextManyReEntries(t *testing.T) {
	expectValue := 10000

	var handlerCounter, middlewareCounter int64

	r := New()
	r.Use(func(c *Context) {
		atomic.AddInt64(&middlewareCounter, 1)
	})
	r.GET("/:count", func(c *Context) {
		countStr := c.Param("count")
		count, err := strconv.Atoi(countStr)
		assert.NoError(t, err)

		n, err := c.Writer.Write([]byte("."))
		assert.NoError(t, err)
		assert.Equal(t, 1, n)

		switch {
		case count > 0:
			c.Request.URL.Path = "/" + strconv.Itoa(count-1)
			r.HandleContext(c)
		}
	}, func(c *Context) {
		atomic.AddInt64(&handlerCounter, 1)
	})

	assert.NotPanics(t, func() {
		w := performRequest(r, "GET", "/"+strconv.Itoa(expectValue-1)) // include 0 value
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, expectValue, w.Body.Len())
	})

	assert.Equal(t, int64(expectValue), handlerCounter)
	assert.Equal(t, int64(expectValue), middlewareCounter)
}

func TestPrepareTrustedCIRDsWith(t *testing.T) {
	r := New()

	// valid ipv4 cidr
	{
		expectedTrustedCIDRs := []*net.IPNet{parseCIDR("0.0.0.0/0")}
		err := r.SetTrustedProxies([]string{"0.0.0.0/0"})

		assert.NoError(t, err)
		assert.Equal(t, expectedTrustedCIDRs, r.trustedCIDRs)
	}

	// invalid ipv4 cidr
	{
		err := r.SetTrustedProxies([]string{"192.168.1.33/33"})

		assert.Error(t, err)
	}

	// valid ipv4 address
	{
		expectedTrustedCIDRs := []*net.IPNet{parseCIDR("192.168.1.33/32")}

		err := r.SetTrustedProxies([]string{"192.168.1.33"})

		assert.NoError(t, err)
		assert.Equal(t, expectedTrustedCIDRs, r.trustedCIDRs)
	}

	// invalid ipv4 address
	{
		err := r.SetTrustedProxies([]string{"192.168.1.256"})

		assert.Error(t, err)
	}

	// valid ipv6 address
	{
		expectedTrustedCIDRs := []*net.IPNet{parseCIDR("2002:0000:0000:1234:abcd:ffff:c0a8:0101/128")}
		err := r.SetTrustedProxies([]string{"2002:0000:0000:1234:abcd:ffff:c0a8:0101"})

		assert.NoError(t, err)
		assert.Equal(t, expectedTrustedCIDRs, r.trustedCIDRs)
	}

	// invalid ipv6 address
	{
		err := r.SetTrustedProxies([]string{"gggg:0000:0000:1234:abcd:ffff:c0a8:0101"})

		assert.Error(t, err)
	}

	// valid ipv6 cidr
	{
		expectedTrustedCIDRs := []*net.IPNet{parseCIDR("::/0")}
		err := r.SetTrustedProxies([]string{"::/0"})

		assert.NoError(t, err)
		assert.Equal(t, expectedTrustedCIDRs, r.trustedCIDRs)
	}

	// invalid ipv6 cidr
	{
		err := r.SetTrustedProxies([]string{"gggg:0000:0000:1234:abcd:ffff:c0a8:0101/129"})

		assert.Error(t, err)
	}

	// valid combination
	{
		expectedTrustedCIDRs := []*net.IPNet{
			parseCIDR("::/0"),
			parseCIDR("192.168.0.0/16"),
			parseCIDR("172.16.0.1/32"),
		}
		err := r.SetTrustedProxies([]string{
			"::/0",
			"192.168.0.0/16",
			"172.16.0.1",
		})

		assert.NoError(t, err)
		assert.Equal(t, expectedTrustedCIDRs, r.trustedCIDRs)
	}

	// invalid combination
	{
		err := r.SetTrustedProxies([]string{
			"::/0",
			"192.168.0.0/16",
			"172.16.0.256",
		})

		assert.Error(t, err)
	}

	// nil value
	{
		err := r.SetTrustedProxies(nil)

		assert.Nil(t, r.trustedCIDRs)
		assert.Nil(t, err)
	}

}

func parseCIDR(cidr string) *net.IPNet {
	_, parsedCIDR, err := net.ParseCIDR(cidr)
	if err != nil {
		fmt.Println(err)
	}
	return parsedCIDR
}

func assertRoutePresent(t *testing.T, gotRoutes RoutesInfo, wantRoute RouteInfo) {
	for _, gotRoute := range gotRoutes {
		if gotRoute.Path == wantRoute.Path && gotRoute.Method == wantRoute.Method {
			assert.Regexp(t, wantRoute.Handler, gotRoute.Handler)
			return
		}
	}
	t.Errorf("route not found: %v", wantRoute)
}

func handlerTest1(c *Context) { c.JSONP(http.StatusOK, "handlerTest1") }
func handlerTestANY(c *Context) {
	c.JSONP(http.StatusOK, "handlerTest any"+c.FullPath()+"===="+c.Request.RequestURI+"--any="+c.Param("any"))
}
func handlerTest2(c *Context) {}
func handlerTest3(c *Context) { c.JSONP(http.StatusOK, "handlerTest1， id="+c.Param("id")) }
func handlerTest4(c *Context) { c.JSONP(http.StatusOK, "handlerTest4") }
func handlerTest5(c *Context) { c.JSONP(http.StatusOK, "handlerTest5") }
func handlerTest6(c *Context) { c.JSONP(http.StatusOK, "handlerTest6,name="+c.Param("name")) }

func TestNew(t *testing.T) {
	router := New()
	fmt.Println("开始前进")
	//router.Handle()
	router.GET("/", handlerTest1)
	group := router.Group("/users")
	{
		group.GET("/", handlerTest2)
		group.GET("/:id", handlerTest1)
		group.GET("/name", handlerTest1)
	}
	router.Run(":90")
}

func TestTree1(t *testing.T) {
	//自定义 debug 信息，开关，是否打印，，默认 debug
	SetMode(DebugMode)
	router := New()
	fmt.Println(IsDebugging())
	fmt.Println(Mode())
	router.Use(Logger(), Recovery())
	fmt.Println("[TestTree1]开始:")
	//router.Handle()
	//router.GET("/", handlerTest1)
	router.Use(middleware1)
	router.Use(middleware2)
	router.GET("/her", handlerTest1)
	router.GET("/her/:id", handlerTest3)
	//router.GET("/her/:name", handlerTest6) //panic: ':name' in new path '/her/:name' conflicts with existing wildcard ':id' in existing prefix '/her/:id' [recovered]
	router.POST("/her/name", handlerTest6)
	router.GET("/his", handlerTest4)
	//router.GET("/his/*", handlerTest)
	router.GET("/his/*any", handlerTestANY)

	router.POST("/her", handlerTest5)

	//fmt.Println(router.Routes())
	router.Run(":90")
}
func middleware1(c *Context) {
	start := time.Now()
	//处理业务
	c.Next()
	elapse := time.Now().Sub(start)
	fmt.Println("[middleware1]处理业务逻辑耗费时间=", elapse)
}

func middleware2(c *Context) {
	fmt.Println("[middleware2] this is middleware2 .")
}

func TestTree5(t *testing.T) {
	//自定义 debug 信息，开关，是否打印，，默认 debug
	SetMode(DebugMode)
	router := New()
	router.Use(Logger(), Recovery())
	fmt.Println("[TestTree5]开始:")
	router.GET("/support", handlerTest1)
	router.GET("/search", handlerTest3)
	router.GET("/blog/:post", handlerTest3)
	router.GET("/about-us", handlerTest3)
	router.GET("/about-us/team", handlerTest3)
	router.GET("/contact", handlerTest3)
	router.Run(":90")
}

//用go的原生http
func TestGoHttp1(t *testing.T) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("  恐龙🦖 "))
	})

	http.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("  恐龙🦖 a "))
	})

	http.HandleFunc("/b", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("  恐龙🦖 b "))
	})

	http.HandleFunc("/b/:id", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("  恐龙🦖 :id "))
	})

	http.HandleFunc("/b/gua", func(w http.ResponseWriter, r *http.Request) {
		panic("故意挂")
	})
	if err := http.ListenAndServe(":80", nil); err != nil {
		log.Fatal("start http server fail:", err)
	}
}

//自定义 实现 ServeHTTP,  给了其它web框架 发挥的空间
type CustomMux struct{}

func (cm *CustomMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "this is xsd \n please enter  www.xueshengduan.com!  ")
	//if r.Method == "GET" {}
}

func TestCustomMux(t *testing.T) {
	log.Fatal(http.ListenAndServe(":80", &CustomMux{}))
}

//设置受信任的代理
func TestEngine_SetTrustedProxies(t *testing.T) {
	r := Default()
	// 下面两种方式2选1即可，推荐使用第二种
	//r.TrustedPlatform = "Client-IP"  // 设置客户端真实ip的请求头
	// 设置受信任的代理
	r.SetTrustedProxies([]string{"127.0.0.1"})
	// 设置url中的大写自动转小写，..和//自动移除，
	r.RedirectFixedPath = true
	// 开启请求方法不允许，并且返回状态码405
	r.HandleMethodNotAllowed = true
	// 设置允许从远程客户端的哪个header头中获取ip（需搭配设置受信任的代理一起使用）
	r.RemoteIPHeaders = append(r.RemoteIPHeaders, "Client-IP")
	// TrustedPlatform 设置可信任的平台，如果增加了此项配置，那么获取客户端真实ip的时候
	// 会优先从请求头中的Real-IP获取，获取到了直接返回，获取不到才会从RemoteIPHeaders中去获取
	// 一般不这样设置，推荐从RemoteIPHeaders中获取，当然前提需要设置受信任的代理, 如果不想设置受信任的代理，那么可以直接从TrustedPlatform中直接获取
	//r.TrustedPlatform = "Real-IP"
	//r.GET("/user/:name", routeUse)
	r.Run(":8000")
}

func TestTree6(t *testing.T) {
	//自定义 debug 信息，开关，是否打印，，默认 debug
	SetMode(DebugMode)
	router := New()
	router.Use(Logger(), Recovery())
	fmt.Println("[TestTree6]开始:")
	router.GET("/support", handlerTest1)
	router.GET("/search", handlerTest3)
	router.Run(":90")
}

func TestTree7(t *testing.T) {
	//自定义 debug 信息，开关，是否打印，，默认 debug
	SetMode(DebugMode)
	router := New()
	router.Use(Logger(), Recovery())
	fmt.Println("[TestTree7]开始:")
	router.GET("/hism", handlerTest1)
	router.GET("/hit", handlerTest3)
	router.GET("/her", handlerTest3)
	router.GET("/havad", handlerTest3)
	router.GET("/has", handlerTest3)
	router.Run(":90")
}

func TestTree8(t *testing.T) {
	//自定义 debug 信息，开关，是否打印，，默认 debug
	SetMode(DebugMode)
	router := New()
	router.Use(Logger(), Recovery())
	fmt.Println("[TestTree8]开始:")
	router.GET("/hism", handlerTest1)
	router.GET("/hit", handlerTest3)
	router.GET("/her", handlerTest3)
	router.GET("/havad", handlerTest3)
	router.GET("/has", handlerTest3)
	router.GET("/has/:id", handlerTest3)
	router.GET("/has/:id/name", handlerTest3)
	router.GET("/has/:id/name/*any", handlerTest3)
	router.Run(":90")
}

func TestTree9(t *testing.T) {
	//自定义 debug 信息，开关，是否打印，，默认 debug
	SetMode(DebugMode)
	router := New()
	fmt.Println("[TestTree9]开始:")
	router.GET("/has/:id", handlerTest3)
	router.GET("/has", handlerTest3)
	router.Run(":90")
}

func TestTree10(t *testing.T) {
	//自定义 debug 信息，开关，是否打印，，默认 debug
	SetMode(DebugMode)
	router := New()
	fmt.Println("[TestTree10]开始:")
	router.GET("/has", handlerTest1)
	router.Run(":90")
}

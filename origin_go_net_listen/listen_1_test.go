package origin_go_net_listen

import (
	"log"
	"net/http"
	"testing"
	"time"
)

func TestListen1(t *testing.T){

	http.HandleFunc("/foo1", func(writer http.ResponseWriter, request *http.Request) {
		_,_ = writer.Write([]byte("this is foo1"))
		return
	})
	log.Fatal(http.ListenAndServe(":80",nil))
	//浏览器输入：http://localhost/foo1
}




type A struct {}
func(A) ServeHTTP (writer http.ResponseWriter, request *http.Request) {
	_,_ = writer.Write([]byte("this is foo2"))
	return
}
func TestListen2(t *testing.T){

	http.Handle("/foo2", &A{})
	log.Fatal(http.ListenAndServe(":80",nil))
	//浏览器输入：http://localhost/foo2
}



type B struct {}
func(B) ServeHTTP (writer http.ResponseWriter, request *http.Request) {
	_,_ = writer.Write([]byte("this is foo3"+request.RequestURI))
	return
}
func TestListen3(t *testing.T){
	h := http.Server{
		Addr:              ":80",
		Handler:           &B{},
		WriteTimeout:      1*time.Second,
		IdleTimeout:       1*time.Second,
		MaxHeaderBytes:    1<<20,
	}
	log.Fatal(h.ListenAndServe())
	//浏览器输入：http://localhost/foo3
}

//https://blog.csdn.net/qq1319713925/article/details/117316349
//https://blog.csdn.net/zrg3699/article/details/122280399

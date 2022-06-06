package main

import (
	"github.com/gin-gonic/gin"

	biz "github.com/sunyakun/thriftgo-tools/example/httpgen"
	"github.com/sunyakun/thriftgo-tools/example/httpgen/http_gen/example"
)

func main() {
	g := gin.Default()
	service := biz.NewExampleService()
	example.Register(g, service)
	err := g.Run(":6789")
	if err != nil {
		panic(err)
	}
}

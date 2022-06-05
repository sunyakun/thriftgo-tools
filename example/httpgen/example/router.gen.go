// Code generated by thriftgo-tools v0.0.1.
package example

import (
	"github.com/gin-gonic/gin"
)

func Register(router gin.IRouter, service ExampleService) {
	handler := NewHandler(service)
	// @route_gen begin
	router.GET("/example/:id", handler.Get)
	router.POST("/example", handler.Create)
	// @route_gen end
}

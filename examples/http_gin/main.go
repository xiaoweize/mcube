package main

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/infraboard/mcube/v2/ioc"
	"github.com/infraboard/mcube/v2/ioc/config/application"
)

func main() {
	// 注册HTTP接口类
	ioc.Api().Registry(&HelloServiceApiHandler{})

	// Ioc配置
	req := ioc.NewLoadConfigRequest()
	err := ioc.ConfigIocObject(req)
	if err != nil {
		panic(err)
	}

	// 启动应用
	err = application.App().Start(context.Background())
	if err != nil {
		panic(err)
	}
}

type HelloServiceApiHandler struct {
	// 继承自Ioc对象
	ioc.ObjectImpl
}

// 模块的名称, 会作为路径的一部分比如: /mcube_service/api/v1/hello_module/
// 路径构成规则 <service_name>/<path_prefix>/<service_version>/<module_name>
func (h *HelloServiceApiHandler) Name() string {
	return "hello_module"
}

func (h *HelloServiceApiHandler) Version() string {
	return "v1"
}

// API路由
func (h *HelloServiceApiHandler) Registry(r gin.IRouter) {
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"data": "hello mcube",
		})
	})
}
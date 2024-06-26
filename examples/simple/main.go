package main

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/infraboard/mcube/v2/ioc"
	"github.com/infraboard/mcube/v2/ioc/config/datasource"
	ioc_gin "github.com/infraboard/mcube/v2/ioc/config/gin"
	"github.com/infraboard/mcube/v2/ioc/server"
	"gorm.io/gorm"

	// 开启Health健康检查
	_ "github.com/infraboard/mcube/v2/ioc/apps/health/gin"
	// 开启Metric
	_ "github.com/infraboard/mcube/v2/ioc/apps/metric/gin"
)

func main() {
	// 注册HTTP接口类
	ioc.Api().Registry(&ApiHandler{})

	// 开启配置文件读取配置
	server.DefaultConfig.ConfigFile.Enabled = true
	server.DefaultConfig.ConfigFile.Path = "etc/application.toml"

	// 启动应用
	err := server.Run(context.Background())
	if err != nil {
		panic(err)
	}
}

type ApiHandler struct {
	// 继承自Ioc对象
	ioc.ObjectImpl

	// mysql db依赖
	db *gorm.DB
}

// 覆写对象的名称, 该名称名称会体现在API的路径前缀里面
// 比如: /simple/api/v1/module_a/db_stats
// 其中/simple/api/v1/module_a 就是对象API前缀, 命名规则如下:
// <service_name>/<path_prefix>/<object_version>/<object_name>
func (h *ApiHandler) Name() string {
	return "module_a"
}

// 初始化db属性, 从ioc的配置区域获取共用工具 gorm db对象
func (h *ApiHandler) Init() error {
	h.db = datasource.DB()

	// 进行业务暴露, router 通过ioc
	router := ioc_gin.RootRouter()
	router.GET("/db_stats", h.GetDbStats)
	return nil
}

// 业务功能
func (h *ApiHandler) GetDbStats(ctx *gin.Context) {
	db, _ := h.db.DB()
	ctx.JSON(http.StatusOK, gin.H{
		"data": db.Stats(),
	})
}

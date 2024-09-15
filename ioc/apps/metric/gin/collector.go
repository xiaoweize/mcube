package metric

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/infraboard/mcube/v2/ioc"
	"github.com/infraboard/mcube/v2/ioc/apps/metric"
	"github.com/infraboard/mcube/v2/ioc/config/application"
	ioc_gin "github.com/infraboard/mcube/v2/ioc/config/gin"
	"github.com/infraboard/mcube/v2/ioc/config/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

func init() {
	ioc.Api().Registry(&apiStatsCollector{})
}

type apiStatsCollector struct {
	log *zerolog.Logger
	ioc.ObjectImpl

	httpRequestTotal    *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
}

func (h *apiStatsCollector) Init() error {
	h.log = log.Sub(metric.AppName)

	h.httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration",
			Help: "Histogram of the duration of HTTP requests",
			ConstLabels: map[string]string{
				"app": application.Get().AppName,
			},
		},
		[]string{"method", "path"},
	)
	h.httpRequestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_request_total",
			Help: "Total number of HTTP rquests",
			ConstLabels: map[string]string{
				"app": application.Get().AppName,
			},
		},
		[]string{"method", "path", "status_code"},
	)
	// 注册采集器
	prometheus.MustRegister(h)

	// 注册中间件
	ioc_gin.RootRouter().Use(h.Middleware())
	return nil
}

func (h *apiStatsCollector) Name() string {
	return "api_stats_collector"
}

func (h *apiStatsCollector) Middleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()

		// 处理请求
		ctx.Next()
		h.httpRequestDuration.WithLabelValues(ctx.Request.Method, ctx.FullPath()).Observe(time.Since(start).Seconds())
		h.httpRequestTotal.WithLabelValues(ctx.Request.Method, ctx.FullPath(), strconv.Itoa(ctx.Writer.Status())).Inc()
	}
}

func (h *apiStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	h.httpRequestDuration.Describe(ch)
	h.httpRequestTotal.Describe(ch)
}

func (h *apiStatsCollector) Collect(ch chan<- prometheus.Metric) {
	h.httpRequestDuration.Collect(ch)
	h.httpRequestTotal.Collect(ch)
}

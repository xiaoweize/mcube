package jsonrpc

import (
	"fmt"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"net/url"
	"reflect"

	"github.com/emicklei/go-restful/v3"
	"github.com/infraboard/mcube/v2/http/restful/response"
	"github.com/infraboard/mcube/v2/ioc"
	"github.com/infraboard/mcube/v2/ioc/config/application"
	"github.com/infraboard/mcube/v2/ioc/config/log"
	"github.com/infraboard/mcube/v2/ioc/config/trace"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful"
)

func init() {
	ioc.Api().Registry(&JsonRpc{
		Host:       "127.0.0.1",
		Port:       9090,
		PathPrefix: "jsonrpc",
	})
}

type JsonRpc struct {
	ioc.ObjectImpl

	// 是否开启HTTP Server, 默认会根据是否有注册得有API对象来自动开启
	Enable *bool `json:"enable" yaml:"enable" toml:"enable" env:"ENABLE"`
	// HTTP服务Host
	Host string `json:"host" yaml:"host" toml:"host" env:"HOST"`
	// HTTP服务端口
	Port int `json:"port" yaml:"port" toml:"port" env:"PORT"`
	// API接口前缀
	PathPrefix string `json:"path_prefix" yaml:"path_prefix" toml:"path_prefix" env:"PATH_PREFIX"`
	// 开启Trace
	Trace bool `toml:"trace" json:"trace" yaml:"trace" env:"TRACE"`
	// 访问日志
	AccessLog bool `toml:"access_log" json:"access_log" yaml:"access_log" env:"ACCESS_LOG"`

	EnableSSL bool   `json:"enable_ssl" yaml:"enable_ssl" toml:"enable_ssl" env:"ENABLE_SSL"`
	CertFile  string `json:"cert_file" yaml:"cert_file" toml:"cert_file" env:"CERT_FILE"`
	KeyFile   string `json:"key_file" yaml:"key_file" toml:"key_file" env:"KEY_FILE"`

	Container *restful.Container
	log       *zerolog.Logger
	services  []service
}

func (h *JsonRpc) Addr() string {
	return fmt.Sprintf("%s:%d", h.Host, h.Port)
}

func (j *JsonRpc) Priority() int {
	return -89
}

func (j *JsonRpc) Name() string {
	return APP_NAME
}

func (h *JsonRpc) HTTPPrefix() string {
	u, err := url.JoinPath("/", h.PathPrefix, application.Get().AppName)
	if err != nil {
		return fmt.Sprintf("/%s/%s", application.Get().AppName, h.PathPrefix)
	}
	return u
}

func (h *JsonRpc) RPCURL() string {
	return fmt.Sprintf("http://%s%s", h.Addr(), h.HTTPPrefix())
}

// 1. 把业务 注册给RPC
func (j *JsonRpc) Registry(name string, svc any) error {
	// 获取 svc 的完整包路径和类型名
	tt := reflect.TypeOf(svc)
	if tt.Kind() == reflect.Ptr {
		tt = tt.Elem()
	}
	fnName := tt.PkgPath() + "." + tt.Name()

	j.services = append(j.services, service{name: name, fnName: fnName})
	return rpc.RegisterName(name, svc)
}

func (j *JsonRpc) Init() error {
	j.log = log.Sub(j.Name())

	if len(j.services) == 0 {
		j.log.Info().Msgf("no reigstry service")
		return nil
	}

	for _, svc := range j.services {
		j.log.Info().Msgf("registe service: %s --> %s", svc.name, svc.fnName)
	}

	j.Container = restful.DefaultContainer
	restful.DefaultResponseContentType(restful.MIME_JSON)
	restful.DefaultRequestContentType(restful.MIME_JSON)

	// 注册路由
	if j.Trace && trace.Get().Enable {
		j.log.Info().Msg("enable jsonrpc trace")
		j.Container.Filter(otelrestful.OTelFilter(application.Get().GetAppName()))
	}

	// RPC的服务架设在“/jsonrpc”路径，
	// 在处理函数中基于http.ResponseWriter和http.Request类型的参数构造一个io.ReadWriteCloser类型的conn通道。
	// 然后基于conn构建针对服务端的json编码解码器。
	// 最后通过rpc.ServeRequest函数为每次请求处理一次RPC方法调用
	ws := new(restful.WebService)
	ws.Path(j.HTTPPrefix()).
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON).
		Route(ws.POST("").To(func(r *restful.Request, w *restful.Response) {
			conn := NewRPCReadWriteCloserFromHTTP(w, r.Request)
			if err := rpc.ServeRequest(jsonrpc.NewServerCodec(conn)); err != nil {
				response.Failed(w, err)
				return
			}
		}))
	// 添加到Root Container
	RootRouter().Add(ws)

	j.log.Info().Msgf("JSON RPC 服务监听地址: %s", j.RPCURL())
	return http.ListenAndServe(j.Addr(), j.Container)
}

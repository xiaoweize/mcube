package ioc

type ObjectStroe interface {
	// 对象注册
	Registry(obj Object)
	// 对象获取
	Get(name string, opts ...GetOption) Object
	// 获取第一个对象
	First() Object
	// 获取最后一个对象
	Last() Object
	// 遍历对象
	ForEach(fn func(Object))
}

// Object 内部服务实例, 不需要暴露
type Object interface {
	// 对象初始化
	Init() error
	// 对象的名称
	Name() string
	// 对象版本
	Version() string
	// 对象优先级
	Priority() int
	// 对象的销毁
	Destory()
	// 是否允许同名对象被替换, 默认不允许被替换
	AllowOverwrite() bool
}

func WithVersion(v string) GetOption {
	return func(o *option) {
		o.version = v
	}
}

type GetOption func(*option)

func defaultOption() *option {
	return &option{
		version: DEFAULT_VERSION,
	}
}

type option struct {
	version string
}

func (o *option) Apply(opts ...GetOption) *option {
	for i := range opts {
		opts[i](o)
	}
	return o
}
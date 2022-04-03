package lol_prophet_gui

type ApplyOption func(o *options)

func WithEnablePprof(enablePprof bool) ApplyOption {
	return func(o *options) {
		o.enablePprof = enablePprof
	}
}

func WithDebug() ApplyOption {
	return func(o *options) {
		o.debug = true
	}
}

func WithProd() ApplyOption {
	return func(o *options) {
		o.debug = false
	}
}

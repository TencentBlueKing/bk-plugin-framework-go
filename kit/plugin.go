package kit

type Plugin interface {
	Version() string
	Desc() string
	Execute(*Context) error
}

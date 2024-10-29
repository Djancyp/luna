package internal

type ReactRoute struct {
	Path  string
	Props func() map[string]interface{}
}

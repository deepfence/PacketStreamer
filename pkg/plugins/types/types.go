package types

type RunningPlugin struct {
	Input  chan<- string
	Errors <-chan error
}

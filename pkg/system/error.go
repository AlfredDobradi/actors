package system

type HandleError interface {
	IsRecoverable() bool
	Error() string
}

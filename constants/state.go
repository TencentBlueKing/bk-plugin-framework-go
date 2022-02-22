package constants

type State int8

const (
	StateEmpty    State = 1
	StatePoll     State = 2
	StateCallback State = 3
	StateSuccess  State = 4
	StateFail     State = 5
)

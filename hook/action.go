package hook

type Action string

const (
	ActionOpen        = "open"
	ActionOpened      = "opened"
	ActionClose       = "close"
	ActionCreate      = "create"
	ActionDelete      = "delete"
	ActionSync        = "sync"
	ActionUpdate      = "update"
	ActionSynchronize = "synchronize"
)

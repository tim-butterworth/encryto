package subscribable

type OutgoingMessage struct {
	Variant string      `json:"variant"`
	Body    interface{} `json:"body"`
}

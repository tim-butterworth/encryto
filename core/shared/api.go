package shared

type Data struct {
	Varient string
	Content interface{}
}

type FromMessage struct {
	From string
	Data Data
}

type ToMessage struct {
	To   string
	Data Data
}

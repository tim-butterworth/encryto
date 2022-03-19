package actors

import "util.tim/encrypto/core/shared"

type IdProvider interface {
	NextId() string
}

type Connection interface {
	Id() string
	Subscribe(func(shared.FromMessage))
	Send(receiver string, message shared.Data)
}

type Exchange interface {
	Connect() Connection
	Reconnect(id string) (Connection, error)
}

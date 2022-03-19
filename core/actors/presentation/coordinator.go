package presentation

import (
	"fmt"
	"sync"

	"util.tim/encrypto/core/actors"
	"util.tim/encrypto/core/shared"
)

type Actions string

const (
	JOIN_NOTES   Actions = "JOIN_NOTES"
	JOIN_PRESENT Actions = "JOIN_PRESENT"
	JOIN_CONTROL Actions = "JOIN_CONTROL"
)

type ActionsMessage struct {
	ActionList []Actions `json:"actions"`
}

type ConcurrentMap struct {
	mutex sync.RWMutex
	ids   map[string]string
}

func NewMemberMap() ConcurrentMap {
	return ConcurrentMap{
		ids: make(map[string]string),
	}
}

func keys(mp map[string]string) []string {
	result := make([]string, len(mp))

	index := 0
	for key := range mp {
		result[index] = key
		index += 1
	}

	return result
}

func (concurrentMap *ConcurrentMap) withReadLock(action func(map[string]string) interface{}) interface{} {
	concurrentMap.mutex.RLock()
	result := action(concurrentMap.ids)
	concurrentMap.mutex.RUnlock()

	return result
}

func (concurrentMap *ConcurrentMap) Ids() []string {
	return concurrentMap.withReadLock(func(m map[string]string) interface{} {
		return keys(concurrentMap.ids)
	}).([]string)
}

func (concurrentMap *ConcurrentMap) Insert(id string) {
	concurrentMap.mutex.Lock()
	concurrentMap.ids[id] = id
	concurrentMap.mutex.Unlock()
}

func (concurrentMap *ConcurrentMap) Contains(id string) bool {
	return concurrentMap.withReadLock(func(m map[string]string) interface{} {
		_, found := m[id]
		return found
	}).(bool)
}

func messageAll(ids []string, connection actors.Connection, message shared.Data) {
	for _, id := range ids {
		connection.Send(id, message)
	}
}

func welcomeMessage(message string) shared.Data {
	return shared.Data{
		Varient: "Message",
		Content: message,
	}
}

func Coordinate(connection actors.Connection) {
	notes := NewMemberMap()
	present := NewMemberMap()
	control := NewMemberMap()

	connection.Subscribe(func(fm shared.FromMessage) {
		fmt.Println(fm.From)
		fmt.Println(fm.Data.Varient)
		fmt.Printf("{%T} -> %s\n", fm.Data.Content, fm.Data.Content)

		if fm.Data.Content == "Actions" {
			connection.Send(fm.From, shared.Data{
				Varient: "Message",
				Content: ActionsMessage{
					ActionList: []Actions{
						JOIN_NOTES,
						JOIN_PRESENT,
						JOIN_CONTROL,
					},
				},
			})
			return
		}

		if fm.Data.Content == string(JOIN_NOTES) {
			fmt.Printf("Adding [%s] to notes group\n", fm.From)
			notes.Insert(fm.From)
			connection.Send(fm.From, welcomeMessage("Welcome to the NOTES Group"))
			return
		}

		if fm.Data.Content == string(JOIN_PRESENT) {
			fmt.Printf("Adding [%s] to present group\n", fm.From)
			present.Insert(fm.From)
			connection.Send(fm.From, welcomeMessage("Welcome to the PRESENT Group"))
			return
		}

		if fm.Data.Content == string(JOIN_CONTROL) {
			fmt.Printf("Adding [%s] to control group\n", fm.From)
			control.Insert(fm.From)
			connection.Send(fm.From, welcomeMessage("Welcome to the CONTROL Group"))
			return
		}

		if control.Contains(fm.From) {
			fmt.Println("Message to all members from CONTROL")
			message := shared.Data{
				Varient: "Message",
				Content: "Notifying about a message from someone",
			}
			messageAll(notes.Ids(), connection, message)
			messageAll(present.Ids(), connection, message)
		}
	})
}

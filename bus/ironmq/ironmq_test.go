package ironmq

import "testing"

func skipTestSend(t *testing.T) {
	// bus := New("dev-queue")

	// msg, err := bus.Receive()
	// if err != nil {
	// 	t.Fatal(err)
	// }
	//
	// t.Log(msg.Header())
	// t.Log(msg.Event())
	// err = bus.Ack(msg)

	// data, err := json.Marshal(struct {
	// 	Key   string
	// 	Value string
	// }{
	// 	Key:   "foo",
	// 	Value: "bar",
	// })
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// raw := json.RawMessage(data)
	//
	// err = bus.Send(eventstore.Event{
	// 	ID:      "poll.1",
	// 	Version: 3,
	// 	Type:    "TestEvent",
	// 	Data:    &raw,
	// })

	// if err != nil {
	// 	t.Fatal(err)
	// }
}

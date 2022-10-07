package p2p

import (
	"fmt"
	"sync"

	"github.com/HPISTechnologies/component-lib/actor"
	intf "github.com/HPISTechnologies/component-lib/interface"
	"github.com/HPISTechnologies/main/modules/p2p/conn/peer"
	"github.com/HPISTechnologies/main/modules/p2p/conn/protocol"
)

type P2pMessage struct {
	Sender  string
	Message *actor.Message
}

var (
	clientInstance *P2pClient
	initClientOnce sync.Once
)

type P2pClient struct {
	actor.WorkerThread

	id        string
	assembler *peer.MessageAssembler
	msgChan   chan *protocol.Message
}

func NewP2pClient(concurrency int, groupId string) actor.IWorkerEx {
	initClientOnce.Do(func() {
		clientInstance = &P2pClient{
			msgChan: make(chan *protocol.Message, 10),
		}
		clientInstance.Set(concurrency, groupId)
		clientInstance.assembler = peer.NewMessageAssembler(clientInstance.msgChan, 10)
		go clientInstance.assembler.Serve()
	})

	return clientInstance
}

func (client *P2pClient) Inputs() ([]string, bool) {
	return []string{
		actor.MsgP2pReceived,
	}, false
}

func (client *P2pClient) Outputs() map[string]int {
	return map[string]int{
		actor.MsgP2pRequest:  100,
		actor.MsgP2pResponse: 100,
		actor.MsgP2pSent:     100,
	}
}

func (client *P2pClient) Config(params map[string]interface{}) {
	client.id = params["cluster_name"].(string)
}

func (client *P2pClient) OnStart() {}

func (client *P2pClient) OnMessageArrived(msgs []*actor.Message) error {
	msg := msgs[0]
	switch msg.Name {
	case actor.MsgP2pReceived:
		// fmt.Printf("[P2pClient.OnMessageArrived] MsgP2pReceived: %v\n", msg)
		// client.NewPackageArrived(msg.Data.([]byte))
		// for _, m := range client.GetReceivedMsg() {
		// 	var msg actor.Message
		// 	msg.Decode(m.Data)
		// 	fmt.Printf("[P2pClient.OnMessageArrived] send msg, name = %s, data = %v\n", msg.Name, msg.Data)
		// 	client.MsgBroker.Send(msg.Name, msg.Data)
		// }
		var m actor.Message
		m.Decode(msg.Data.(*protocol.Message).Data)
		fmt.Printf("[P2pClient.OnMessageArrived] send msg, name = %s, data = %v\n", m.Name, m.Data)
		client.MsgBroker.Send(m.Name, m.Data)
	}
	return nil
}

func (client *P2pClient) ID() string {
	return client.id
}

func (client *P2pClient) Broadcast(msg *actor.Message) {
	fmt.Printf("[P2pClient.Broadcast] msg = %v\n", msg)
	data, err := (&actor.Message{
		Msgid: msg.Msgid,
		Name:  actor.MsgP2pRequest,
		Data: &P2pMessage{
			Sender:  client.id,
			Message: msg,
		},
	}).Encode()
	if err != nil {
		panic(err)
	}

	packages := protocol.Message{
		ID:   msg.Msgid,
		Type: protocol.MessageTypeClientBroadcast,
		Data: data,
	}.ToPackages()
	fmt.Printf("[P2pClient.Broadcast] ready to send %d packages\n", len(packages))
	for _, p := range packages {
		b, _ := p.MarshalBinary()
		client.MsgBroker.Send(actor.MsgP2pSent, b)
	}
}

func (client *P2pClient) Request(peer string, msg *actor.Message) {
	data, err := (&actor.Message{
		Msgid: msg.Msgid,
		Name:  actor.MsgP2pRequest,
		Data: &P2pMessage{
			Sender:  client.id,
			Message: msg,
		},
	}).Encode()
	if err != nil {
		panic(err)
	}

	fmt.Printf("[P2pClient.Request] send data, len = %d\n", len(data))
	var na int
	intf.Router.Call("p2p", "Send", &P2pSendRequest{
		Peer: peer,
		Msg: &protocol.Message{
			ID:   msg.Msgid,
			Type: protocol.MessageTypeClientSend,
			Data: data,
		},
	}, &na)
}

func (client *P2pClient) Response(peer string, msg *actor.Message) {
	data, err := (&actor.Message{
		Msgid: msg.Msgid,
		Name:  actor.MsgP2pResponse,
		Data: &P2pMessage{
			Sender:  client.id,
			Message: msg,
		},
	}).Encode()
	if err != nil {
		panic(err)
	}

	fmt.Printf("[P2pClient.Response] send data, len = %d\n", len(data))
	var na int
	intf.Router.Call("p2p", "Send", &P2pSendRequest{
		Peer: peer,
		Msg: &protocol.Message{
			ID:   msg.Msgid,
			Type: protocol.MessageTypeClientSend,
			Data: data,
		},
	}, &na)
}

func (client *P2pClient) OnConnClosed(cb func(id string)) {}

func (client *P2pClient) NewPackageArrived(data []byte) {
	var p protocol.Package
	p.UnmarshalBinary(data)
	p.Body = data[protocol.PackageHeaderSize:]
	fmt.Printf("[P2pClient.NewPackageArrived] package header = %v, len(body) = %d\n", p.Header, len(p.Body))

	if p.Header.TotalPackageCount == 1 {
		var m protocol.Message
		client.msgChan <- m.FromPackages([]*protocol.Package{&p})
	} else {
		client.assembler.AddPart(&p)
	}
}

func (client *P2pClient) GetReceivedMsg() []*protocol.Message {
	var msgs []*protocol.Message
	for {
		select {
		case m := <-client.msgChan:
			msgs = append(msgs, m)
		default:
			fmt.Printf("[P2pClient.GetReceivedMsg] return %d messages\n", len(msgs))
			return msgs
		}
	}
}

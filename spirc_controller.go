package stringutil

import (
    "github.com/golang/protobuf/proto"
    "github.com/badfortrains/Spotify"
    "fmt"
    "log"
)

type controller struct{
	session *Session
	seqNr uint32
	ident string
	username string
	devices map[string]connectDevice
}

type connectDevice struct{
    name string
    ident string
}


func SetupController(session *Session, username string, ident string) controller{
	return controller{
		devices: make(map[string]connectDevice),
		session: session,
		username: username,
		ident: ident,
	}
}

func (c *controller) sendHello(){

}

func (c *controller) sendCmd(recipient []string, messageType Spotify.MessageType) {
	c.seqNr += 1
	frame := &Spotify.Frame{
		Version: proto.Uint32(1),
		Ident: proto.String(c.ident),
		ProtocolVersion: proto.String("2.0.0"),
		SeqNr: proto.Uint32(c.seqNr),
		Typ: &messageType,
		Recipient: recipient,
	}

	frameData, err := proto.Marshal(frame)
	if err != nil {
		log.Fatal("could not Marshal request frame")
	}

	payload := make([][]byte,1)
	payload[0] = frameData

	c.session.MercurySendRequest(MercuryRequest{
			method: "SUB",
			uri: "hm://remote/3/user/" + c.username + "/",
			payload: payload,
		}, nil)
}

func (c *controller) run(){
	ch := make(chan MercuryResponse)
	c.session.MercurySubscribe("hm://remote/3/user/" + c.username + "/", ch)

	for {
		reponse :=  <- ch

		frame := &Spotify.Frame{}
		err := proto.Unmarshal(reponse.payload[0], frame)
		if err != nil {
			fmt.Println("error getting packet") 
			continue
		}

		if frame.GetTyp() == Spotify.MessageType_kMessageTypeHello {
			c.devices[*frame.Ident] = connectDevice{
				name: *frame.DeviceState.Name,
				ident: *frame.Ident,
			}
		}

		fmt.Printf("%v %v %v %v %v %v \n",
			frame.Typ,
			*frame.DeviceState.Name,
			*frame.Ident,
			*frame.SeqNr,
			*frame.StateUpdateId,
			frame.Recipient,
		)

	}

}
package spotcontrol

import (
	"fmt"
	Spotify "github.com/badfortrains/spotcontrol/proto"
	"github.com/golang/protobuf/proto"
	"log"
	"sync"
)

type SpircController struct {
	session  *Session
	seqNr    uint32
	ident    string
	username string
	devices  map[string]connectDevice
	devicesLock sync.RWMutex
}

type connectDevice struct {
	Name  string
	Ident string
}

func SetupController(session *Session, username string, ident string) SpircController {
	return SpircController{
		devices:  make(map[string]connectDevice),
		session:  session,
		username: username,
		ident:    ident,
	}
}

func (c *SpircController) LoadTrack(ident string, gids []string) {
	c.seqNr += 1

	tracks := make([]*Spotify.TrackRef,0, len(gids))
	for _, g := range gids{
		tracks = append(tracks, &Spotify.TrackRef{
			Gid: Convert62(g),
			Queued: proto.Bool(false),
		})
	}

	state := &Spotify.State{
		Index:             proto.Uint32(0),
		Track:             tracks,
		Status:            Spotify.PlayStatus_kPlayStatusStop.Enum(),
		PlayingTrackIndex: proto.Uint32(0),
	}

	frame := &Spotify.Frame{
		Version:         proto.Uint32(1),
		Ident:           proto.String(c.ident),
		ProtocolVersion: proto.String("2.0.0"),
		SeqNr:           proto.Uint32(c.seqNr),
		Typ:             Spotify.MessageType_kMessageTypeLoad.Enum(),
		Recipient:       []string{ident},
		State:           state,
	}

	c.sendFrame(frame)
}

func (c *SpircController) SendHello() {
	c.sendCmd(nil, Spotify.MessageType_kMessageTypeHello)
}

func (c *SpircController) SendPlay(ident string) {
	c.sendCmd([]string{ident}, Spotify.MessageType_kMessageTypePlay)
}

func (c *SpircController) SendPause(ident string) {

	c.sendCmd([]string{ident}, Spotify.MessageType_kMessageTypePause)
}

func (c *SpircController) ListDevices() []connectDevice {
	c.devicesLock.RLock()
	res := make([]connectDevice, 0, len(c.devices))
	for _, device := range c.devices {
		res = append(res, device)
	}
	c.devicesLock.RUnlock()
	return res
}

func (c *SpircController) sendFrame(frame *Spotify.Frame) {
	frameData, err := proto.Marshal(frame)
	if err != nil {
		log.Fatal("could not Marshal request frame")
	}

	payload := make([][]byte, 1)
	payload[0] = frameData

	c.session.MercurySendRequest(MercuryRequest{
		method:  "SEND",
		uri:     "hm://remote/user/" + c.username + "/v23",
		payload: payload,
	}, nil)
}

func (c *SpircController) sendCmd(recipient []string, messageType Spotify.MessageType) {
	c.seqNr += 1
	frame := &Spotify.Frame{
		Version:         proto.Uint32(1),
		Ident:           proto.String(c.ident),
		ProtocolVersion: proto.String("2.0.0"),
		SeqNr:           proto.Uint32(c.seqNr),
		Typ:             &messageType,
		Recipient:       recipient,
	}

	c.sendFrame(frame)
}

func (c *SpircController) Run() {
	ch := make(chan MercuryResponse)
	c.session.MercurySubscribe("hm://remote/user/"+c.username+"/v23", ch)

	for {
		reponse := <-ch

		frame := &Spotify.Frame{}
		err := proto.Unmarshal(reponse.payload[0], frame)
		if err != nil {
			fmt.Println("error getting packet")
			continue
		}

		if frame.GetTyp() == Spotify.MessageType_kMessageTypeNotify {
			c.devicesLock.Lock()
			c.devices[*frame.Ident] = connectDevice{
				Name:  frame.DeviceState.GetName(),
				Ident: *frame.Ident,
			}
			c.devicesLock.Unlock()
		}

		fmt.Printf("%v %v %v %v %v %v \n",
			frame.Typ,
			frame.DeviceState.GetName(),
			*frame.Ident,
			*frame.SeqNr,
			frame.GetStateUpdateId(),
			frame.Recipient,
		)

	}

}

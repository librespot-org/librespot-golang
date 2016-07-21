package spotcontrol

import (
	"fmt"
	Spotify "github.com/badfortrains/spotcontrol/proto"
	"github.com/golang/protobuf/proto"
	"strings"
	"sync"
)

type SpircController struct {
	session     *session
	seqNr       uint32
	ident       string
	username    string
	devices     map[string]ConnectDevice
	devicesLock sync.RWMutex
	updateChan  chan Spotify.Frame

	SavedCredentials []byte
}

// Represents an available spotify connect device.
// For mdns devices not yet authenitcated, Ident will be ""
// and Url will be the address to pass to ConnectToDevie.
type ConnectDevice struct {
	Name   string
	Ident  string
	Url    string
	Volume int
}

// Starts controller.  Registers listeners for Spotify connect device
// updates, and opens connection for sending commands
func setupController(userSession *session, username string, credentials []byte) *SpircController {
	if username == "" &&
		userSession.discovery.loginBlob.Username != "" {
		username = userSession.discovery.loginBlob.Username
	}

	controller := &SpircController{
		devices:          make(map[string]ConnectDevice),
		session:          userSession,
		username:         username,
		ident:            userSession.deviceId,
		SavedCredentials: credentials,
	}
	controller.subscribe()
	return controller
}

// Load comma seperated tracks
func (c *SpircController) LoadTrackIds(ident string, ids string) error {
	return c.LoadTrack(ident, strings.Split(ids, ","))
}

// Load given list of tracks on spotify connect device with given
// ident.  Gids are formated base62 spotify ids.
func (c *SpircController) LoadTrack(ident string, gids []string) error {
	c.seqNr += 1

	tracks := make([]*Spotify.TrackRef, 0, len(gids))
	for _, g := range gids {
		tracks = append(tracks, &Spotify.TrackRef{
			Gid:    convert62(g),
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

	return c.sendFrame(frame)
}

// Sends a 'hello' command to all spotify connect devices.
// Active devices will respond with a 'notify' updating
// their state.
func (c *SpircController) SendHello() error {
	return c.sendCmd(nil, Spotify.MessageType_kMessageTypeHello)
}

// Sends a 'play' command to spotify connect device with
// given ident.
func (c *SpircController) SendPlay(ident string) error {
	return c.sendCmd([]string{ident}, Spotify.MessageType_kMessageTypePlay)
}

// Sends a 'pause' command to spotify connect device with
// given ident.
func (c *SpircController) SendPause(ident string) error {
	return c.sendCmd([]string{ident}, Spotify.MessageType_kMessageTypePause)
}

func (c *SpircController) SendVolume(ident string, volume int) error {
	c.seqNr += 1
	messageType := Spotify.MessageType_kMessageTypeVolume
	frame := &Spotify.Frame{
		Version:         proto.Uint32(1),
		Ident:           proto.String(c.ident),
		ProtocolVersion: proto.String("2.0.0"),
		SeqNr:           proto.Uint32(c.seqNr),
		Typ:             &messageType,
		Recipient:       []string{ident},
		Volume:          proto.Uint32(uint32(volume)),
	}

	return c.sendFrame(frame)
}

// Connect to spotify-connect device at address (local network path).
// Uses credentials from saved blob to authenticate.
func (c *SpircController) ConnectToDevice(address string) {
	c.session.discovery.ConnectToDevice(address)
}

// Lists devices on local network advertising spotify connect
// service (_spotify-connect._tcp.).
func (c *SpircController) ListMdnsDevices() []ConnectDevice {
	discovery := c.session.discovery
	discovery.devicesLock.RLock()
	res := make([]ConnectDevice, 0, len(discovery.devices))
	for _, device := range discovery.devices {
		res = append(res, ConnectDevice{
			Name: device.name,
			Url:  device.path,
		})
	}
	discovery.devicesLock.RUnlock()
	return res
}

// List active spotify-connect devices that can be sent commands
func (c *SpircController) ListDevices() []ConnectDevice {
	c.devicesLock.RLock()
	res := make([]ConnectDevice, 0, len(c.devices))
	for _, device := range c.devices {
		res = append(res, device)
	}
	c.devicesLock.RUnlock()

	return res
}

func (c *SpircController) sendFrame(frame *Spotify.Frame) error {
	frameData, err := proto.Marshal(frame)
	if err != nil {
		return fmt.Errorf("could not Marshal spirc request frame: ", err)
	}

	payload := make([][]byte, 1)
	payload[0] = frameData

	status := make(chan int32)

	go c.session.mercurySendRequest(mercuryRequest{
		method:  "SEND",
		uri:     "hm://remote/user/" + c.username + "/",
		payload: payload,
	}, func(res mercuryResponse) {
		status <- res.statusCode
	})

	code := <-status
	if code >= 200 && code < 300 {
		return nil
	} else {
		return fmt.Errorf("spirc send frame got mercury response, status code: %v", code)
	}
}

func (c *SpircController) sendCmd(recipient []string, messageType Spotify.MessageType) error {
	c.seqNr += 1
	frame := &Spotify.Frame{
		Version:         proto.Uint32(1),
		Ident:           proto.String(c.ident),
		ProtocolVersion: proto.String("2.0.0"),
		SeqNr:           proto.Uint32(c.seqNr),
		Typ:             &messageType,
		Recipient:       recipient,
	}

	return c.sendFrame(frame)
}

func (c *SpircController) subscribe() {
	ch := make(chan mercuryResponse)
	c.session.mercurySubscribe("hm://remote/user/"+c.username+"/", ch, func(_ mercuryResponse) {
		go c.run(ch)
		go c.SendHello()
	})
}

func (c *SpircController) run(ch chan mercuryResponse) {
	for {
		reponse := <-ch

		frame := &Spotify.Frame{}
		err := proto.Unmarshal(reponse.payload[0], frame)
		if err != nil {
			fmt.Println("error getting packet")
			continue
		}

		if frame.GetTyp() == Spotify.MessageType_kMessageTypeNotify ||
			(frame.GetTyp() == Spotify.MessageType_kMessageTypeHello && frame.DeviceState.GetName() != "") {
			c.devicesLock.Lock()
			c.devices[*frame.Ident] = ConnectDevice{
				Name:   frame.DeviceState.GetName(),
				Ident:  *frame.Ident,
				Volume: int(frame.DeviceState.GetVolume()),
			}
			c.devicesLock.Unlock()
		} else if frame.GetTyp() == Spotify.MessageType_kMessageTypeGoodbye {
			c.devicesLock.Lock()
			delete(c.devices, *frame.Ident)
			c.devicesLock.Unlock()
		}

		if c.updateChan != nil {
			select {
			case c.updateChan <- *frame:
				fmt.Println("sent update")
			default:
				fmt.Println("dropped update")
			}
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

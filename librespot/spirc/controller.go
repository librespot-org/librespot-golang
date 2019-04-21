package spirc

import (
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/librespot-org/librespot-golang/Spotify"
	"github.com/librespot-org/librespot-golang/librespot/core"
	"github.com/librespot-org/librespot-golang/librespot/mercury"
	"github.com/librespot-org/librespot-golang/librespot/utils"
	"strings"
	"sync"
)

// Controller is a structure for Spotify Connect remote control interface.
type Controller struct {
	session     *core.Session
	seqNr       uint32
	devices     map[string]ConnectDevice
	devicesLock sync.RWMutex
	updateChan  chan Spotify.Frame

	SavedCredentials []byte
}

// Represents an available Spotify connect device.
// For mdns devices not yet authenticated, Ident will be ""
// and Url will be the address to pass to ConnectToDevice.
type ConnectDevice struct {
	Name   string
	Ident  string
	Url    string
	Volume int
}

// CreateController creates a Spirc controller. Registers listeners for Spotify connect device
// updates, and opens connection for sending commands
func CreateController(userSession *core.Session, credentials []byte) *Controller {
	controller := &Controller{
		devices:          make(map[string]ConnectDevice),
		session:          userSession,
		SavedCredentials: credentials,
	}
	controller.subscribe()
	return controller
}

// Load comma seperated tracks
func (c *Controller) LoadTrackIds(ident string, ids string) error {
	return c.LoadTrack(ident, strings.Split(ids, ","))
}

// Load given list of tracks on spotify connect device with given
// ident.  Gids are formated base62 spotify ids.
func (c *Controller) LoadTrack(ident string, gids []string) error {
	c.seqNr += 1

	tracks := make([]*Spotify.TrackRef, 0, len(gids))
	for _, g := range gids {
		tracks = append(tracks, &Spotify.TrackRef{
			Gid:    utils.Convert62(g),
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
		Ident:           proto.String(c.session.DeviceId()),
		ProtocolVersion: proto.String("2.0.0"),
		SeqNr:           proto.Uint32(c.seqNr),
		Typ:             Spotify.MessageType_kMessageTypeLoad.Enum(),
		Recipient:       []string{ident},
		State:           state,
	}

	return c.sendFrame(frame)
}

// Sends a 'hello' command to all Spotify Connect devices. Active devices will respond with a 'notify' updating
// their state.
func (c *Controller) SendHello() error {
	return c.sendCmd(nil, Spotify.MessageType_kMessageTypeHello)
}

// Sends a 'play' command to spotify connect device with given identity (recipient param).
func (c *Controller) SendPlay(recipient string) error {
	return c.sendCmd([]string{recipient}, Spotify.MessageType_kMessageTypePlay)
}

// Sends a 'pause' command to Spotify Connect device with given identity (recipient param).
func (c *Controller) SendPause(recipient string) error {
	return c.sendCmd([]string{recipient}, Spotify.MessageType_kMessageTypePause)
}

func (c *Controller) SendVolume(recipient string, volume int) error {
	c.seqNr += 1
	messageType := Spotify.MessageType_kMessageTypeVolume
	frame := &Spotify.Frame{
		Version:         proto.Uint32(1),
		Ident:           proto.String(c.session.DeviceId()),
		ProtocolVersion: proto.String("2.0.0"),
		SeqNr:           proto.Uint32(c.seqNr),
		Typ:             &messageType,
		Recipient:       []string{recipient},
		Volume:          proto.Uint32(uint32(volume)),
	}

	return c.sendFrame(frame)
}

// Connect to Spotify Connect device at address (local network path). Uses credentials from saved blob to authenticate
// on the device automagically.
func (c *Controller) ConnectToDevice(address string) {
	c.session.Discovery().ConnectToDevice(address)
}

// Lists devices on local network advertising spotify connect
// service (_spotify-connect._tcp.).
func (c *Controller) ListMdnsDevices() ([]ConnectDevice, error) {
	discovery := c.session.Discovery()
	if discovery == nil {
		return nil, errors.New(
			"no connectDiscovery blob, must load blob before getting mdns devices")
	}

	devices := discovery.Devices()
	res := make([]ConnectDevice, 0, len(devices))
	for _, device := range devices {
		res = append(res, ConnectDevice{
			Name: device.Name,
			Url:  device.Path,
		})
	}

	return res, nil
}

// List active spotify-connect devices that can be sent commands
func (c *Controller) ListDevices() []ConnectDevice {
	c.devicesLock.RLock()
	res := make([]ConnectDevice, 0, len(c.devices))
	for _, device := range c.devices {
		res = append(res, device)
	}
	c.devicesLock.RUnlock()

	return res
}

func (c *Controller) sendFrame(frame *Spotify.Frame) error {
	frameData, err := proto.Marshal(frame)
	if err != nil {
		return fmt.Errorf("could not Marshal spirc Request frame: ", err)
	}

	payload := make([][]byte, 1)
	payload[0] = frameData

	status := make(chan int32)

	go c.session.Mercury().Request(mercury.Request{
		Method:  "SEND",
		Uri:     "hm://remote/user/" + c.session.Username() + "/",
		Payload: payload,
	}, func(res mercury.Response) {
		status <- res.StatusCode
	})

	code := <-status
	if code >= 200 && code < 300 {
		return nil
	} else {
		return fmt.Errorf("spirc send frame got mercury response, status code: %v", code)
	}
}

func (c *Controller) sendCmd(recipient []string, messageType Spotify.MessageType) error {
	c.seqNr += 1
	frame := &Spotify.Frame{
		Version:         proto.Uint32(1),
		Ident:           proto.String(c.session.DeviceId()),
		ProtocolVersion: proto.String("2.0.0"),
		SeqNr:           proto.Uint32(c.seqNr),
		Typ:             &messageType,
		Recipient:       recipient,
	}

	return c.sendFrame(frame)
}

func (c *Controller) subscribe() {
	ch := make(chan mercury.Response)
	c.session.Mercury().Subscribe(fmt.Sprintf("hm://remote/user/%s/", c.session.Username()), ch, func(_ mercury.Response) {
		go c.run(ch)
		go c.SendHello()
	})
}

func (c *Controller) run(ch chan mercury.Response) {
	for {
		response := <-ch

		frame := &Spotify.Frame{}
		err := proto.Unmarshal(response.Payload[0], frame)
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

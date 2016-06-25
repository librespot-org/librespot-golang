package spotcontrol

import (
	"encoding/json"
	"fmt"
	Spotify "github.com/badfortrains/spotcontrol/proto"
)

type Updater interface {
	OnUpdate(device string)
}

func (c *SpircController) HandleUpdates(u Updater) {
	c.updateChan = make(chan Spotify.Frame, 5)

	go func() {
		for {
			update := <-c.updateChan
			json, err := json.Marshal(update)
			if err != nil {
				fmt.Println("Error marhsaling device json")
			} else {
				u.OnUpdate(string(json))
			}
		}
	}()
}

func (c *SpircController) ListDevicesJson() (string, error) {
	devices := c.ListDevices()
	json, err := json.Marshal(devices)
	if err != nil {
		return "", nil
	}
	return string(json), nil
}

func (c *SpircController) ListMdnsDevicesJson() (string, error) {
	devices := c.ListMdnsDevices()
	json, err := json.Marshal(devices)
	if err != nil {
		return "", nil
	}
	return string(json), nil
}

package librespotmobile

import (
	"librespot/player"
	"librespot/core"
)

// MobilePlayer is a gomobile-compliant subset of the Player struct.
type MobilePlayer struct {
	player *player.Player
}

func createMobilePlayer(session *core.Session) *MobilePlayer {
	return &MobilePlayer{
		player: session.Player(),
	}
}



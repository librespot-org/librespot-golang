package librespotmobile

import (
	"github.com/librespot-org/librespot-golang/Spotify"
	"github.com/librespot-org/librespot-golang/librespot/core"
	"github.com/librespot-org/librespot-golang/librespot/player"
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

func (p *MobilePlayer) LoadTrack(fileId []byte, format int, trackId []byte) (*MobileAudioFile, error) {
	// Make a copy of the fileId and trackId byte arrays, as they may be freed/reused on the other end,
	// causing the fileId and/or trackId to change abruptly when the player actually request chunks.
	safeFileId := make([]byte, len(fileId))
	safeTrackId := make([]byte, len(trackId))
	copy(safeFileId, fileId)
	copy(safeTrackId, trackId)

	track, err := p.player.LoadTrackWithIdAndFormat(safeFileId, Spotify.AudioFile_Format(format), safeTrackId)
	if err != nil {
		return nil, err
	}

	return createMobileAudioFile(track), nil
}

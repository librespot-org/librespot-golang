package librespotmobile

import "github.com/librespot-org/librespot-golang/librespot/player"

// MobileAudioFile is a gomobile-compliant subset of the AudioFile struct. It
// is allocated by the MobilePlayer struct and functions.
type MobileAudioFile struct {
	audioFile *player.AudioFile
}

func createMobileAudioFile(file *player.AudioFile) *MobileAudioFile {
	return &MobileAudioFile{
		audioFile: file,
	}
}

func (a *MobileAudioFile) Size() int32 {
	return int32(a.audioFile.Size())
}

func (a *MobileAudioFile) Read(buf []byte) (int, error) {
	return a.audioFile.Read(buf)
}

func (a *MobileAudioFile) Seek(offset int32, whence int) (int, error) {
	pos, err := a.audioFile.Seek(int64(offset), whence)
	return int(pos), err
}

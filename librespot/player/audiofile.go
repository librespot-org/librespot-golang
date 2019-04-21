package player

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"github.com/librespot-org/librespot-golang/Spotify"
	"github.com/librespot-org/librespot-golang/librespot/connection"
	"io"
	"math"
	"sync"
)

const kChunkSize = 32768 // In number of words (so actual byte size is kChunkSize*4, aka. kChunkByteSize)
const kChunkByteSize = kChunkSize * 4
const kOggSkipBytes = 167 // Number of bytes to skip at the beginning of the file

// min helper function for integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// AudioFile represents a downloadable/cached audio file fetched by Spotify, in an encoded format (OGG, etc)
type AudioFile struct {
	size           uint32
	lock           sync.RWMutex
	format         Spotify.AudioFile_Format
	fileId         []byte
	player         *Player
	cipher         cipher.Block
	decrypter      *AudioFileDecrypter
	responseChan   chan []byte
	chunkLock      sync.RWMutex
	chunkLoadOrder []int
	data           []byte
	cursor         int
	chunks         map[int]bool
	chunksLoading  bool
}

func newAudioFile(file *Spotify.AudioFile, player *Player) *AudioFile {
	return newAudioFileWithIdAndFormat(file.GetFileId(), file.GetFormat(), player)
}

func newAudioFileWithIdAndFormat(fileId []byte, format Spotify.AudioFile_Format, player *Player) *AudioFile {
	return &AudioFile{
		player:        player,
		fileId:        fileId,
		format:        format,
		decrypter:     NewAudioFileDecrypter(),
		size:          kChunkSize, // Set an initial size to fetch the first chunk regardless of the actual size
		responseChan:  make(chan []byte),
		chunks:        map[int]bool{},
		chunkLock:     sync.RWMutex{},
		chunksLoading: false,
	}
}

// Size returns the size, in bytes, of the final audio file
func (a *AudioFile) Size() uint32 {
	return a.size - uint32(a.headerOffset())
}

// Read is an implementation of the io.Reader interface. Note that due to the nature of the streaming, we may return
// zero bytes when we are waiting for audio data from the Spotify servers, so make sure to wait for the io.EOF error
// before stopping playback.
func (a *AudioFile) Read(buf []byte) (int, error) {
	length := len(buf)
	outBufCursor := 0
	totalWritten := 0
	eof := false

	a.lock.RLock()
	size := a.size
	a.lock.RUnlock()
	// Offset the data start by the header, if needed
	if a.cursor == 0 {
		a.cursor += a.headerOffset()
	} else if uint32(a.cursor) >= size {
		// We're at the end
		return 0, io.EOF
	}

	// Ensure at least the next required chunk is fully available, otherwise request and wait for it. Even if we
	// don't have the entire data required for len(buf) (because it overlaps two or more chunks, and only the first
	// one is available), we can still return the data already available, we don't need to wait to fill the entire
	// buffer.
	chunkIdx := a.chunkIndexAtByte(a.cursor)

	for totalWritten < length {
		// fmt.Printf("[audiofile] Cursor: %d, len: %d, matching chunk %d\n", a.cursor, length, chunkIdx)

		if chunkIdx >= a.totalChunks() {
			// We've reached the last chunk, so we can signal EOF
			eof = true
			break
		} else if !a.hasChunk(chunkIdx) {
			// A chunk we are looking to read is unavailable, request it so that we can return it on the next Read call
			a.requestChunk(chunkIdx)
			// fmt.Printf("[audiofile] Doesn't have chunk %d yet, queuing\n", chunkIdx)
			break
		} else {
			// cursorEnd is the ending position in the output buffer. It is either the current outBufCursor + the size
			// of a chunk, in bytes, or the length of the buffer, whichever is smallest.
			cursorEnd := min(outBufCursor+kChunkByteSize, length)
			writtenLen := cursorEnd - outBufCursor

			// Calculate where our data cursor will end: either at the boundary of the current chunk, or the end
			// of the song itself
			dataCursorEnd := min(a.cursor+writtenLen, (chunkIdx+1)*kChunkByteSize)
			dataCursorEnd = min(dataCursorEnd, int(a.size))

			writtenLen = dataCursorEnd - a.cursor

			if writtenLen <= 0 {
				// No more space in the output buffer, bail out
				break
			}

			// Copy into the output buffer, from the current outBufCursor, up to the cursorEnd. The source is the
			// current cursor inside the audio file, up to the dataCursorEnd.
			copy(buf[outBufCursor:cursorEnd], a.data[a.cursor:dataCursorEnd])
			outBufCursor += writtenLen
			a.cursor += writtenLen
			totalWritten += writtenLen

			// Update our current chunk, if we need to
			chunkIdx = a.chunkIndexAtByte(a.cursor)
		}
	}

	// The only error we can return here, is if we reach the end of the stream
	var err error
	if eof {
		err = io.EOF
	}

	return totalWritten, err
}

// Seek implements the io.Seeker interface
func (a *AudioFile) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		a.cursor = int(offset) + a.headerOffset()

	case io.SeekEnd:
		a.cursor = int(int64(a.size) + offset)

	case io.SeekCurrent:
		a.cursor += int(offset)
	}

	return int64(a.cursor - a.headerOffset()), nil
}

func (a *AudioFile) headerOffset() int {
	// If the file format is an OGG, we skip the first kOggSkipBytes (167) bytes. We could implement despotify's
	// SpotifyOggHeader (https://sourceforge.net/p/despotify/code/HEAD/tree/java/trunk/src/main/java/se/despotify/client/player/SpotifyOggHeader.java)
	// to read Spotify's metadata (samples, length, gain, ...). For now, we simply skip the custom header to the actual
	// OGG/Vorbis data.
	switch {
	case a.format == Spotify.AudioFile_OGG_VORBIS_96 || a.format == Spotify.AudioFile_OGG_VORBIS_160 ||
		a.format == Spotify.AudioFile_OGG_VORBIS_320:
		return kOggSkipBytes

	default:
		return 0
	}
}

func (a *AudioFile) chunkIndexAtByte(byteIndex int) int {
	return int(math.Floor(float64(byteIndex) / float64(kChunkSize) / 4.0))
}

func (a *AudioFile) hasChunk(index int) bool {
	a.chunkLock.RLock()
	has, ok := a.chunks[index]
	a.chunkLock.RUnlock()

	return has && ok
}

func (a *AudioFile) loadKey(trackId []byte) error {
	key, err := a.player.loadTrackKey(trackId, a.fileId)
	if err != nil {
		fmt.Printf("[audiofile] Unable to load key: %s\n", err)
		return err
	}

	a.cipher, err = aes.NewCipher(key)
	if err != nil {
		return err
	}

	return nil
}

func (a *AudioFile) totalChunks() int {
	a.lock.RLock()
	size := a.size
	a.lock.RUnlock()
	return int(math.Ceil(float64(size) / float64(kChunkSize) / 4.0))
}

func (a *AudioFile) loadChunks() {
	// By default, we will load the track in the normal order. If we need to skip to a specific piece of audio,
	// we will prepend the chunks needed so that we load them as soon as possible. Since loadNextChunk will check
	// if a chunk is already loaded (using hasChunk), we won't be downloading the same chunk multiple times.

	// We can however only download the first chunk for now, as we have no idea how many chunks this track has. The
	// remaining chunks will be added once we get the headers with the file size.
	a.chunkLoadOrder = append(a.chunkLoadOrder, 0)

	go a.loadNextChunk()
}

func (a *AudioFile) requestChunk(chunkIndex int) {
	a.chunkLock.RLock()

	// Check if we don't already have this chunk in the 2 next chunks requested
	if len(a.chunkLoadOrder) >= 1 && a.chunkLoadOrder[0] == chunkIndex ||
		len(a.chunkLoadOrder) >= 2 && a.chunkLoadOrder[1] == chunkIndex {
		a.chunkLock.RUnlock()
		return
	}

	a.chunkLock.RUnlock()

	// Set an artificial limit of 500 chunks to prevent overflows and buggy readers/seekers
	a.chunkLock.Lock()

	if len(a.chunkLoadOrder) < 500 {
		a.chunkLoadOrder = append([]int{chunkIndex}, a.chunkLoadOrder...)
	}

	a.chunkLock.Unlock()
}

func (a *AudioFile) loadChunk(chunkIndex int) error {
	chunkData := make([]byte, kChunkByteSize)

	channel := a.player.AllocateChannel()
	channel.onHeader = a.onChannelHeader
	channel.onData = a.onChannelData

	chunkOffsetStart := uint32(chunkIndex * kChunkSize)
	chunkOffsetEnd := uint32((chunkIndex + 1) * kChunkSize)
	err := a.player.stream.SendPacket(connection.PacketStreamChunk, buildAudioChunkRequest(channel.num, a.fileId, chunkOffsetStart, chunkOffsetEnd))

	if err != nil {
		return err
	}

	chunkSz := 0

	for {
		chunk := <-a.responseChan
		chunkLen := len(chunk)

		if chunkLen > 0 {
			copy(chunkData[chunkSz:chunkSz+chunkLen], chunk)
			chunkSz += chunkLen

			// fmt.Printf("Read %d/%d of chunk %d\n", sz, expSize, i)
		} else {
			break
		}
	}

	// fmt.Printf("[AudioFile] Got encrypted chunk %d, len=%d...\n", i, len(wholeData))

	a.putEncryptedChunk(chunkIndex, chunkData[0:chunkSz])

	return nil

}

func (a *AudioFile) loadNextChunk() {
	a.chunkLock.Lock()

	if a.chunksLoading {
		// We are already loading a chunk, don't need to start another goroutine
		a.chunkLock.Unlock()
		return
	}

	a.chunksLoading = true
	chunkIndex := a.chunkLoadOrder[0]
	a.chunkLoadOrder = a.chunkLoadOrder[1:]

	a.chunkLock.Unlock()

	if !a.hasChunk(chunkIndex) {
		a.loadChunk(chunkIndex)
	}

	a.chunkLock.Lock()
	a.chunksLoading = false

	if len(a.chunkLoadOrder) > 0 {
		a.chunkLock.Unlock()
		a.loadNextChunk()
	} else {
		a.chunkLock.Unlock()
	}
}

func (a *AudioFile) putEncryptedChunk(index int, data []byte) {
	byteIndex := index * kChunkByteSize
	a.decrypter.DecryptAudioWithBlock(index, a.cipher, data, a.data[byteIndex:byteIndex+len(data)])

	a.chunkLock.Lock()
	a.chunks[index] = true
	a.chunkLock.Unlock()
}

func (a *AudioFile) onChannelHeader(channel *Channel, id byte, data *bytes.Reader) uint16 {
	read := uint16(0)

	if id == 0x3 {
		var size uint32
		binary.Read(data, binary.BigEndian, &size)
		size *= 4
		// fmt.Printf("[AudioFile] Audio file size: %d bytes\n", size)

		if a.size != size {
			a.lock.Lock()
			a.size = size
			a.lock.Unlock()
			if a.data == nil {
				a.data = make([]byte, size)
			}

			// Recalculate the number of chunks pending for load
			a.chunkLock.Lock()
			for i := 0; i < a.totalChunks(); i++ {
				a.chunkLoadOrder = append(a.chunkLoadOrder, i)
			}
			a.chunkLock.Unlock()

			// Re-launch the chunk loading system. It will check itself if another goroutine is already loading chunks.
			go a.loadNextChunk()
		}

		// Return 4 bytes read
		read = 4
	}

	return read
}

func (a *AudioFile) onChannelData(channel *Channel, data []byte) uint16 {
	if data != nil {
		a.responseChan <- data

		return 0 // uint16(len(data))
	} else {
		// fmt.Printf("[AudioFile] Got EOF (nil) audio data on channel %d!\n", channel.num)
		a.responseChan <- []byte{}

		return 0
	}

}

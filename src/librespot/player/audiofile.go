package player

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"math"
)

const kChunkSize = 32768 // In number of words (so actual byte size is kChunkSize*4)

type AudioFile struct {
	Size         uint32
	Chunks       map[int]bool
	Data         []byte
	FileId       []byte
	Player       *Player
	Decrypter    *AudioFileDecrypter
	Cipher       cipher.Block
	responseChan chan []byte
}

func NewAudioFile(fileId []byte, key []byte, player *Player) *AudioFile {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	return &AudioFile{
		Player:       player,
		FileId:       fileId,
		Cipher:       block,
		Decrypter:    NewAudioFileDecrypter(),
		Size:         kChunkSize, // Set an initial size to fetch the first chunk regardless of the actual size
		responseChan: make(chan []byte),
		Chunks:       map[int]bool{},
	}
}

func (a *AudioFile) Load() error {
	// Request audio data

	chunkData := make([]byte, kChunkSize*4)

	for i := 0; i < a.TotalChunks(); i++ {
		fmt.Printf("[audiofile] Requesting chunk %d...\n", i)
		channel := a.Player.AllocateChannel()
		channel.onHeader = a.onChannelHeader
		channel.onData = a.onChannelData

		chunkOffsetStart := uint32(i * kChunkSize)
		chunkOffsetEnd := uint32((i + 1) * kChunkSize)
		err := a.Player.stream.SendPacket(0x8, buildAudioChunkRequest(channel.num, a.FileId, chunkOffsetStart, chunkOffsetEnd))

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

		// fmt.Printf("[audiofile] Got encrypted chunk %d, len=%d...\n", i, len(wholeData))

		a.PutEncryptedChunk(i, chunkData[0:chunkSz])
	}

	// OGG header fixup
	if a.Data[5] == 0x06 {
		a.Data[5] = 0x02
	}

	fmt.Printf("[audiofile] Loaded %d chunks\n", a.TotalChunks())

	return nil
}

func (a *AudioFile) HasChunk(index int) bool {
	has, ok := a.Chunks[index]
	return has && ok
}

func (a *AudioFile) TotalChunks() int {
	return int(math.Ceil(float64(a.Size) / float64(kChunkSize) / 4.0))
}

func (a *AudioFile) PutEncryptedChunk(index int, data []byte) {
	byteIndex := index * kChunkSize * 4
	a.Decrypter.DecryptAudioWithBlock(index, a.Cipher, data, a.Data[byteIndex:byteIndex+len(data)])
	a.Chunks[index] = true
}

func (a *AudioFile) onChannelHeader(channel *Channel, id byte, data *bytes.Reader) uint16 {
	read := uint16(0)

	if id == 0x3 {
		var size uint32
		binary.Read(data, binary.BigEndian, &size)
		size *= 4
		// fmt.Printf("[audiofile] Audio file size: %d bytes\n", size)

		a.Size = size
		if a.Data == nil {
			a.Data = make([]byte, size)
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
		// fmt.Printf("[audiofile] Got EOF (nil) audio data on channel %d!\n", channel.num)
		a.responseChan <- []byte{}

		return 0
	}

}

func buildAudioChunkRequest(channel uint16, fileId []byte, start uint32, end uint32) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, channel)
	binary.Write(buf, binary.BigEndian, uint8(0x0))
	binary.Write(buf, binary.BigEndian, uint8(0x1))
	binary.Write(buf, binary.BigEndian, uint16(0x0000))
	binary.Write(buf, binary.BigEndian, uint32(0x00000000))
	binary.Write(buf, binary.BigEndian, uint32(0x00009C40))
	binary.Write(buf, binary.BigEndian, uint32(0x00020000))
	buf.Write(fileId)
	binary.Write(buf, binary.BigEndian, start)
	binary.Write(buf, binary.BigEndian, end)

	return buf.Bytes()
}

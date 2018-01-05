package player

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"librespot/connection"
	"librespot/mercury"
	"librespot/utils"
	"log"
)

type Player struct {
	stream    connection.PacketStream
	mercury   *mercury.Client
	seq       uint32
	audioKey  []byte
	readyChan chan bool

	channels map[uint16]*Channel
	nextChan uint16
}

func CreatePlayer(conn connection.PacketStream, client *mercury.Client) *Player {
	return &Player{
		stream:    conn,
		mercury:   client,
		readyChan: make(chan bool),
		channels:  map[uint16]*Channel{},
		nextChan:  0,
	}
}

func (p *Player) LoadTrack(trackId []byte, fileId []byte) (*AudioFile, error) {
	fmt.Printf("[player] Loading track audio key, fileId: %s, trackId: %s\n", utils.ConvertTo62(fileId), utils.ConvertTo62(trackId))
	fmt.Printf("[player] Track as hex: %x\nFile as hex: %x\n", trackId, fileId)

	err := p.stream.SendPacket(0xc, p.buildKeyRequest(trackId, fileId))

	if err != nil {
		log.Println("Error while sending packet", err)
	}

	<-p.readyChan

	log.Println("[player] Fetching audio data")

	// Allocate a channel
	channel := p.AllocateChannel()

	// Allocate an AudioFile
	audioFile := NewAudioFile(fileId, channel, p.stream)
	err = audioFile.Load()

	return audioFile, err
}

func (p *Player) AllocateChannel() *Channel {
	channel := NewChannel(p.nextChan, p.releaseChannel)
	channel.onHeader = headerFunc(p.onChannelHeader)
	channel.onData = dataFunc(p.onChannelData)
	p.nextChan++

	p.channels[channel.num] = channel
}

func (p *Player) HandleCmd(cmd byte, data []byte) {
	switch {
	case cmd == 0xd:
		// Audio key response
		p.audioKey = data[8:]

		// Request audio data
		p.readyChan <- true

	case cmd == 0xe:
		// Audio key error
		fmt.Println("[player] Audio key error!")
		fmt.Printf("%x\n", data)

	case cmd == 0x9:
		// Audio data response
		var channel uint16
		dataReader := bytes.NewReader(data)
		binary.Read(dataReader, binary.BigEndian, &channel)

		fmt.Printf("[player] Data on channel %d: %d bytes\n", channel, len(data[2:]))

		if val, ok := p.channels[channel]; ok {
			val.handlePacket(data[2:])
		} else {
			fmt.Printf("Unknown channel!\n")
		}
	}
}

func (p *Player) buildKeyRequest(trackId []byte, fileId []byte) []byte {
	buf := new(bytes.Buffer)

	buf.Write(fileId)
	buf.Write(trackId)
	buf.Write(p.mercury.NextSeq())
	binary.Write(buf, binary.BigEndian, uint16(0x0000))

	return buf.Bytes()
}

func (p *Player) releaseChannel(channel *Channel) {
	delete(p.channels, channel.num)
	fmt.Printf("[player] Released channel %d\n", channel.num)
}

func (p *Player) onChannelHeader(channel *Channel, id byte, data *bytes.Reader) uint16 {
	read := uint16(0)

	if id == 0x3 {
		var size uint32
		binary.Read(data, binary.BigEndian, &size)
		fmt.Printf("[player] Audio file size: %d bytes\n", size)

		// Return 4 bytes read
		read = 4
	}

	return read
}

func (p *Player) onChannelData(channel *Channel, data *bytes.Reader) uint16 {
	if data != nil {
		fmt.Printf("[player] Got audio channel data!\n")
	} else {
		fmt.Printf("[player] Got EOF (nil) audio data!\n")
	}
	return 0
}

package player

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"librespot/connection"
	"librespot/mercury"
	"librespot/utils"
	"log"
	"sync"
)

type Player struct {
	stream   connection.PacketStream
	mercury  *mercury.Client
	seq      uint32
	audioKey []byte
	keyChan  chan []byte

	chanLock sync.Mutex
	channels map[uint16]*Channel
	nextChan uint16
}

func CreatePlayer(conn connection.PacketStream, client *mercury.Client) *Player {
	return &Player{
		stream:   conn,
		mercury:  client,
		keyChan:  make(chan []byte),
		channels: map[uint16]*Channel{},
		chanLock: sync.Mutex{},
		nextChan: 0,
	}
}

func (p *Player) LoadTrack(trackId []byte, fileId []byte) (*AudioFile, error) {
	fmt.Printf("[player] Loading track audio key, fileId: %s, trackId: %s\n", utils.ConvertTo62(fileId), utils.ConvertTo62(trackId))
	fmt.Printf("[player] Track as hex: %x\nFile as hex: %x\n", trackId, fileId)

	err := p.stream.SendPacket(0xc, p.buildKeyRequest(trackId, fileId))

	if err != nil {
		log.Println("Error while sending packet", err)
	}

	key := <-p.keyChan

	log.Printf("[player] Got key %x, fetching audio data\n", key)

	// Allocate an AudioFile and a channel
	channel := p.AllocateChannel()
	audioFile := NewAudioFile(fileId, key, p)

	channel.onHeader = headerFunc(audioFile.onChannelHeader)
	channel.onData = dataFunc(audioFile.onChannelData)

	// Start loading the audio
	err = audioFile.Load()

	return audioFile, err
}

func (p *Player) AllocateChannel() *Channel {
	p.chanLock.Lock()
	channel := NewChannel(p.nextChan, p.releaseChannel)
	p.nextChan++

	p.channels[channel.num] = channel
	p.chanLock.Unlock()

	return channel
}

func (p *Player) HandleCmd(cmd byte, data []byte) {
	switch {
	case cmd == 0xd:
		// Audio key response
		p.keyChan <- data[4:20]

	case cmd == 0xe:
		// Audio key error
		fmt.Println("[player] Audio key error!")
		fmt.Printf("%x\n", data)

	case cmd == 0x9:
		// Audio data response
		var channel uint16
		dataReader := bytes.NewReader(data)
		binary.Read(dataReader, binary.BigEndian, &channel)

		// fmt.Printf("[player] Data on channel %d: %d bytes\n", channel, len(data[2:]))

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
	p.chanLock.Lock()
	delete(p.channels, channel.num)
	p.chanLock.Unlock()
	fmt.Printf("[player] Released channel %d\n", channel.num)
}

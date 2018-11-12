package player

import (
	"Spotify"
	"bytes"
	"encoding/binary"
	"fmt"
	"librespot/connection"
	"librespot/mercury"
	"log"
	"sync"
)

type Player struct {
	stream   connection.PacketStream
	mercury  *mercury.Client
	seq      uint32
	audioKey []byte

	chanLock sync.Mutex
	seqChanLock sync.Mutex
	channels map[uint16]*Channel
	seqChans map[uint32]chan []byte
	nextChan uint16
}

func CreatePlayer(conn connection.PacketStream, client *mercury.Client) *Player {
	return &Player{
		stream:   conn,
		mercury:  client,
		channels: map[uint16]*Channel{},
		seqChans: map[uint32]chan []byte{},
		chanLock: sync.Mutex{},
		seqChanLock: sync.Mutex{},
		nextChan: 0,
	}
}

func (p *Player) LoadTrack(file *Spotify.AudioFile, trackId []byte) (*AudioFile, error) {
	return p.LoadTrackWithIdAndFormat(file.FileId, file.GetFormat(), trackId)
}

func (p *Player) LoadTrackWithIdAndFormat(fileId []byte, format Spotify.AudioFile_Format, trackId []byte) (*AudioFile, error) {
	// fmt.Printf("[player] Loading track audio key, fileId: %s, trackId: %s\n", utils.ConvertTo62(fileId), utils.ConvertTo62(trackId))

	// Allocate an AudioFile and a channel
	audioFile := newAudioFileWithIdAndFormat(fileId, format, p)

	// Start loading the audio key
	err := audioFile.loadKey(trackId)

	// Then start loading the audio itself
	audioFile.loadChunks()

	return audioFile, err
}

func (p *Player) loadTrackKey(trackId []byte, fileId []byte) ([]byte, error) {
	seqInt, seq := p.mercury.NextSeqWithInt()

	p.seqChanLock.Lock()
	p.seqChans[seqInt] = make(chan []byte)
	p.seqChanLock.Unlock()

	req := buildKeyRequest(seq, trackId, fileId)
	err := p.stream.SendPacket(connection.PacketRequestKey, req)
	if err != nil {
		log.Println("Error while sending packet", err)
		return nil, err
	}

	p.seqChanLock.Lock()
	key := <-p.seqChans[seqInt]
	delete(p.seqChans, seqInt)
	p.seqChanLock.Unlock()

	return key, nil
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
	case cmd == connection.PacketAesKey:
		// Audio key response
		dataReader := bytes.NewReader(data)
		var seqNum uint32
		binary.Read(dataReader, binary.BigEndian, &seqNum)

		p.seqChanLock.Lock()
		if channel, ok := p.seqChans[seqNum]; ok {
			channel <- data[4:20]
		} else {
			fmt.Printf("[player] Unknown channel for audio key seqNum %d\n", seqNum)
		}
		p.seqChanLock.Unlock()

	case cmd == connection.PacketAesKeyError:
		// Audio key error
		fmt.Println("[player] Audio key error!")
		fmt.Printf("%x\n", data)

	case cmd == connection.PacketStreamChunkRes:
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

func (p *Player) releaseChannel(channel *Channel) {
	p.chanLock.Lock()
	delete(p.channels, channel.num)
	p.chanLock.Unlock()
	// fmt.Printf("[player] Released channel %d\n", channel.num)
}

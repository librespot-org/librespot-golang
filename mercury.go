package spotcontrol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	Spotify "github.com/badfortrains/spotcontrol/proto"
	"github.com/golang/protobuf/proto"
	"io"
	"sync"
)

type mercuryResponse struct {
	headerData []byte
	uri        string
	payload    [][]byte
	statusCode int32
	seqKey     string
}

type mercuryRequest struct {
	method      string
	uri         string
	contentType string
	payload     [][]byte
}

type responseCallback func(mercuryResponse)

type mercuryPending struct {
	parts   [][]byte
	partial []byte
}

type mercuryInternal struct {
	seqLock sync.Mutex
	nextSeq uint32
	pending map[string]mercuryPending
	stream  packetStream
}

type mercuryClient struct {
	subscriptions map[string][]chan mercuryResponse
	callbacks     map[string]responseCallback
	mInternal     *mercuryInternal
}

func setupMercury(s *session) mercuryCon {
	client := &mercuryClient{
		callbacks:     make(map[string]responseCallback),
		subscriptions: make(map[string][]chan mercuryResponse),
		mInternal: &mercuryInternal{
			pending: make(map[string]mercuryPending),
			stream:  s.stream,
		},
	}
	return client
}

func (m *mercuryClient) addChanelSubscriber(uri string, recv chan mercuryResponse) {
	chList, ok := m.subscriptions[uri]
	if !ok {
		chList = make([]chan mercuryResponse, 0)
	}

	chList = append(chList, recv)
	m.subscriptions[uri] = chList
}

func (m *mercuryClient) Subscribe(uri string, recv chan mercuryResponse, cb responseCallback) error {
	m.addChanelSubscriber(uri, recv)
	err := m.request(mercuryRequest{
		method: "SUB",
		uri:    uri,
	}, func(response mercuryResponse) {
		for _, part := range response.payload {
			sub := &Spotify.Subscription{}
			err := proto.Unmarshal(part, sub)
			if err == nil && *sub.Uri != uri {
				m.addChanelSubscriber(*sub.Uri, recv)
			}
		}
		cb(response)
	})

	return err
}

func (m *mercuryClient) request(req mercuryRequest, cb responseCallback) (err error) {
	seq, err := m.mInternal.request(req)
	if err != nil {
		return err
	}
	m.callbacks[string(seq)] = cb
	return nil
}

func (m *mercuryInternal) request(req mercuryRequest) (seqKey string, err error) {
	m.seqLock.Lock()
	seq := make([]byte, 4)
	binary.BigEndian.PutUint32(seq, m.nextSeq)
	m.nextSeq += 1
	m.seqLock.Unlock()
	data, err := encodeRequest(seq, req)
	if err != nil {
		return "", err
	}

	var cmd uint8
	switch {
	case req.method == "SUB":
		cmd = 0xb3
	case req.method == "UNSUB":
		cmd = 0xb4
	default:
		cmd = 0xb2
	}

	err = m.stream.SendPacket(cmd, data)
	if err != nil {
		return "", err
	}

	return string(seq), nil
}

func encodeMercuryHead(seq []byte, partsLength uint16, flags uint8) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, uint16(len(seq)))
	if err != nil {
		return nil, err
	}
	_, err = buf.Write(seq)
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.BigEndian, uint8(flags))
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.BigEndian, partsLength)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func encodeRequest(seq []byte, req mercuryRequest) ([]byte, error) {
	buf, err := encodeMercuryHead(seq, uint16(1+len(req.payload)), uint8(1))
	if err != nil {
		return nil, err
	}

	header := &Spotify.Header{
		Uri:    proto.String(req.uri),
		Method: proto.String(req.method),
	}

	if req.contentType != "" {
		header.ContentType = proto.String(req.contentType)
	}

	headerData, err := proto.Marshal(header)
	err = binary.Write(buf, binary.BigEndian, uint16(len(headerData)))
	if err != nil {
		return nil, err
	}
	_, err = buf.Write(headerData)
	if err != nil {
		return nil, err
	}

	for _, p := range req.payload {
		err = binary.Write(buf, binary.BigEndian, uint16(len(p)))
		if err != nil {
			return nil, err
		}
		_, err = buf.Write(p)
		if err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func handleHead(reader io.Reader) (seq []byte, flags uint8, count uint16, err error) {
	var seqLength uint16
	err = binary.Read(reader, binary.BigEndian, &seqLength)
	if err != nil {
		return
	}
	seq = make([]byte, seqLength)
	_, err = io.ReadFull(reader, seq)
	if err != nil {
		fmt.Println("read seq")
		return
	}

	err = binary.Read(reader, binary.BigEndian, &flags)
	if err != nil {
		fmt.Println("read flags")
		return
	}
	err = binary.Read(reader, binary.BigEndian, &count)
	if err != nil {
		fmt.Println("read count")
		return
	}

	return
}

func (m *mercuryClient) handle(cmd uint8, reader io.Reader) (err error) {
	response, err := m.mInternal.parseResponse(cmd, reader)
	if err != nil {
		return
	}
	if response != nil {
		if cmd == 0xb5 {
			chList, ok := m.subscriptions[response.uri]
			if ok {
				for _, ch := range chList {
					ch <- *response
				}
			}
		} else if cb, ok := m.callbacks[response.seqKey]; ok {
			delete(m.callbacks, response.seqKey)
			cb(*response)
		}
	}
	return

}

func (m *mercuryInternal) parseResponse(cmd uint8, reader io.Reader) (response *mercuryResponse, err error) {
	seq, flags, count, err := handleHead(reader)
	if err != nil {
		fmt.Println("error handling response", err)
		return
	}

	seqKey := string(seq)
	pending, ok := m.pending[seqKey]

	if !ok && cmd == 0xb5 {
		pending = mercuryPending{}
	} else if !ok {
		//log.Print("ignoring seq ", seqKey)
	}

	for i := uint16(0); i < count; i++ {
		part, err := parsePart(reader)
		if err != nil {
			fmt.Println("read part")
			return nil, err
		}

		if pending.partial != nil {
			part = append(pending.partial, part...)
			pending.partial = nil
		}

		if i == count-1 && (flags == 2) {
			pending.partial = part
		} else {
			pending.parts = append(pending.parts, part)
		}
	}

	if flags == 1 {
		delete(m.pending, seqKey)
		return m.completeRequest(cmd, pending, seqKey)
	} else {
		m.pending[seqKey] = pending
	}
	return nil, nil
}

func (m *mercuryInternal) completeRequest(cmd uint8, pending mercuryPending, seqKey string) (response *mercuryResponse, err error) {
	headerData := pending.parts[0]
	header := &Spotify.Header{}
	err = proto.Unmarshal(headerData, header)
	if err != nil {
		return nil, err
	}

	return &mercuryResponse{
		headerData: headerData,
		uri:        *header.Uri,
		payload:    pending.parts[1:],
		statusCode: header.GetStatusCode(),
		seqKey:     seqKey,
	}, nil

}

func parsePart(reader io.Reader) ([]byte, error) {
	var size uint16
	binary.Read(reader, binary.BigEndian, &size)
	buf := make([]byte, size)
	_, err := io.ReadFull(reader, buf)
	return buf, err
}

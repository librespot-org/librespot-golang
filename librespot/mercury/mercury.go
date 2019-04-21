package mercury

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/librespot-org/librespot-golang/Spotify"
	"github.com/librespot-org/librespot-golang/librespot/connection"
	"io"
	"sync"
)

// Mercury is the protocol implementation for Spotify Connect playback control and metadata fetching.It works as a
// PUB/SUB system, where you, as an audio sink, subscribes to the events of a specified user (playlist changes) but
// also access various metadata normally fetched by external players (tracks metadata, playlists, artists, etc).

type Response struct {
	HeaderData []byte
	Uri        string
	Payload    [][]byte
	StatusCode int32
	SeqKey     string
}

type Request struct {
	Method      string
	Uri         string
	ContentType string
	Payload     [][]byte
}

type Callback func(Response)

type Pending struct {
	parts   [][]byte
	partial []byte
}

type Internal struct {
	seqLock sync.Mutex
	nextSeq uint32
	pending map[string]Pending
	stream  connection.PacketStream
}

type Client struct {
	subscriptions map[string][]chan Response
	callbacks     map[string]Callback
	internal      *Internal
	cbMu          sync.Mutex
}

type Connection interface {
	Subscribe(uri string, recv chan Response, cb Callback) error
	Request(req Request, cb Callback) (err error)
	Handle(cmd uint8, reader io.Reader) (err error)
}

// CreateMercury initializes a Connection for the specified session.
func CreateMercury(stream connection.PacketStream) *Client {
	client := &Client{
		callbacks:     make(map[string]Callback),
		subscriptions: make(map[string][]chan Response),
		internal: &Internal{
			pending: make(map[string]Pending),
			stream:  stream,
		},
	}
	return client
}

// Subscribe subscribes the specified receiving channel to the specified URI, and calls the callback function
// whenever there's an event happening.
func (m *Client) Subscribe(uri string, recv chan Response, cb Callback) error {
	m.addChannelSubscriber(uri, recv)
	err := m.Request(Request{
		Method: "SUB",
		Uri:    uri,
	}, func(response Response) {
		for _, part := range response.Payload {
			sub := &Spotify.Subscription{}
			err := proto.Unmarshal(part, sub)
			if err == nil && *sub.Uri != uri {
				m.addChannelSubscriber(*sub.Uri, recv)
			}
		}
		cb(response)
	})

	return err
}

func (m *Client) addChannelSubscriber(uri string, recv chan Response) {
	chList, ok := m.subscriptions[uri]
	if !ok {
		chList = make([]chan Response, 0)
	}

	chList = append(chList, recv)
	m.subscriptions[uri] = chList
}

func (m *Client) Request(req Request, cb Callback) (err error) {
	seq, err := m.internal.request(req)
	if err != nil {
		// Call the callback with a 500 error-code so that the request doesn't remain pending in case of error
		if cb != nil {
			cb(Response{
				StatusCode: 500,
			})
		}

		return err
	}

	m.cbMu.Lock()
	m.callbacks[string(seq)] = cb
	m.cbMu.Unlock()

	return nil
}

func (m *Client) NextSeq() []byte {
	_, seq := m.internal.NextSeq()
	return seq
}

func (m *Client) NextSeqWithInt() (uint32, []byte) {
	return m.internal.NextSeq()
}

func (m *Internal) NextSeq() (uint32, []byte) {
	m.seqLock.Lock()

	seq := make([]byte, 4)
	seqInt := m.nextSeq
	binary.BigEndian.PutUint32(seq, seqInt)
	m.nextSeq += 1
	m.seqLock.Unlock()

	return seqInt, seq
}

func (m *Internal) request(req Request) (seqKey string, err error) {
	_, seq := m.NextSeq()
	data, err := encodeRequest(seq, req)
	if err != nil {
		return "", err
	}

	var cmd uint8
	switch {
	case req.Method == "SUB":
		cmd = 0xb3
	case req.Method == "UNSUB":
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

func encodeRequest(seq []byte, req Request) ([]byte, error) {
	buf, err := encodeMercuryHead(seq, uint16(1+len(req.Payload)), uint8(1))
	if err != nil {
		return nil, err
	}

	header := &Spotify.Header{
		Uri:    proto.String(req.Uri),
		Method: proto.String(req.Method),
	}

	if req.ContentType != "" {
		header.ContentType = proto.String(req.ContentType)
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

	for _, p := range req.Payload {
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

func (m *Client) Handle(cmd uint8, reader io.Reader) (err error) {
	response, err := m.internal.parseResponse(cmd, reader)
	if err != nil {
		return
	}
	if response != nil {
		if cmd == 0xb5 {
			chList, ok := m.subscriptions[response.Uri]
			if ok {
				for _, ch := range chList {
					ch <- *response
				}
			}
		} else {
			m.cbMu.Lock()
			cb, ok := m.callbacks[response.SeqKey]
			delete(m.callbacks, response.SeqKey) // no-op if element does not exist
			m.cbMu.Unlock()
			if ok {
				cb(*response)
			}
		}
	}
	return

}

func (m *Internal) parseResponse(cmd uint8, reader io.Reader) (response *Response, err error) {
	seq, flags, count, err := handleHead(reader)
	if err != nil {
		fmt.Println("error handling response", err)
		return
	}

	seqKey := string(seq)
	pending, ok := m.pending[seqKey]

	if !ok && cmd == 0xb5 {
		pending = Pending{}
	} else if !ok {
		//log.Print("ignoring seq ", SeqKey)
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

func (m *Internal) completeRequest(cmd uint8, pending Pending, seqKey string) (response *Response, err error) {
	headerData := pending.parts[0]
	header := &Spotify.Header{}
	err = proto.Unmarshal(headerData, header)
	if err != nil {
		return nil, err
	}

	return &Response{
		HeaderData: headerData,
		Uri:        *header.Uri,
		Payload:    pending.parts[1:],
		StatusCode: header.GetStatusCode(),
		SeqKey:     seqKey,
	}, nil

}

func parsePart(reader io.Reader) ([]byte, error) {
	var size uint16
	binary.Read(reader, binary.BigEndian, &size)
	buf := make([]byte, size)
	_, err := io.ReadFull(reader, buf)
	return buf, err
}

func (res *Response) CombinePayload() []byte {
	body := make([]byte, 0)
	for _, p := range res.Payload {
		body = append(body, p...)
	}
	return body
}

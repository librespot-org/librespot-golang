package stringutil

import (
    "io"
    "encoding/binary"
    "log"
    "github.com/golang/protobuf/proto"
    "github.com/badfortrains/Spotify"
    "fmt"
    "bytes"
)

const (
    Mercury_GET = iota
    Mercury_SUB
    Mercury_UNSUB
    Mercury_SEND
)

type MercuryResponse struct{
    uri string
    payload [][]byte
}

type MercuryRequest struct{
    method string
    uri string
    contentType string
    payload [][]byte
}

type MercuryPending struct{
    parts [][]byte
    partial []byte
    result []chan MercuryResponse
}

type MercuryManager struct{
    nextSeq uint32
    pending map[string]MercuryPending
    subscriptions map[string][]chan MercuryResponse
    session *Session
}

func SetupMercury(s *Session) MercuryManager{
    return MercuryManager{
        pending: make(map[string] MercuryPending),
        subscriptions: make(map[string][]chan MercuryResponse),
        session: s,
    }
}

func (m *MercuryManager) Subscribe(uri string, recv chan MercuryResponse) (error){
    chList, ok := m.subscriptions[uri]
    if !ok {
        chList = make([]chan MercuryResponse, 0)
    }

    chList = append(chList, recv)
    m.subscriptions[uri] = chList

    err := m.request(MercuryRequest{
        method: "SUB",
        uri: uri,
    }, nil)

    return err
}

func (m *MercuryManager) request(req MercuryRequest, resultCh chan MercuryResponse) (err error){
    seq := make([]byte, 4)
    binary.BigEndian.PutUint32(seq, m.nextSeq)
    m.nextSeq += 1
    data, err := encodeRequest(seq, req)
    if err != nil {
        return err
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

    m.session.stream.SendPacket(cmd, data)
    return nil
}

func encodeRequest(seq []byte, req MercuryRequest) ([]byte, error){
    buf := new(bytes.Buffer)
    err := binary.Write(buf, binary.BigEndian, uint16(len(seq)))
    if err != nil {
        return nil , err
    }
    _, err = buf.Write(seq)
    if err != nil {
        return nil , err
    }
    err = binary.Write(buf, binary.BigEndian, uint8(1))
    if err != nil {
        return nil , err
    }
    err = binary.Write(buf, binary.BigEndian, 
        uint16(1 + len(req.payload)))
    if err != nil {
        return nil , err
    }

    header := &Spotify.Header{
        Uri: proto.String(req.uri),
        Method: proto.String(req.method),
    }

    if req.contentType != "" {
        header.ContentType = proto.String(req.contentType)
    }

    headerData, err := proto.Marshal(header)
    err = binary.Write(buf, binary.BigEndian, uint16(len(headerData)))
    if err != nil {
        return nil , err
    }
    _, err = buf.Write(headerData)
    if err != nil {
        return nil , err
    }

    for _, p := range req.payload {
        err = binary.Write(buf, binary.BigEndian, uint16(len(p)))
        if err != nil {
            return nil , err
        }
        _, err = buf.Write(p)
        if err != nil {
            return nil , err
        }
    }

    return buf.Bytes(), nil
}

func (m *MercuryManager) handle(cmd uint8, reader io.Reader) (err error){
    var seqLength uint16
    var flags uint8
    var count uint16

    err = binary.Read(reader, binary.BigEndian, &seqLength)
    if err != nil {
        return
    }
    seq := make([]byte, seqLength)
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

    seqKey := string(seq)
    pending, ok := m.pending[seqKey];

    if !ok && cmd == 0xb5{
        pending = MercuryPending{}
    } else if !ok {
        log.Print("ignoring seq ", seqKey)
    }

    for i := uint16(0); i < count; i++ {
        part, err := parsePart(reader)
        if err != nil {
            fmt.Println("read part")
            return err
        }

        if pending.partial != nil {
            part = append(part, pending.partial...)
            pending.partial = nil
        }

        if i == count - 1 && (flags == 2) {
            pending.partial = part
        } else {
            pending.parts = append(pending.parts,part)
        }
    }

    if flags == 1 {
        m.completeRequest(cmd, pending)
    } else {
        m.pending[seqKey] = pending
    }
    return
}

func (m *MercuryManager) completeRequest(cmd uint8, pending MercuryPending) (err error){
    headerData := pending.parts[0]
    header := &Spotify.Header{}
    err = proto.Unmarshal(headerData, header)
    if err != nil {
        return err
    }

    response := MercuryResponse{
        uri: *header.Uri,
        payload: pending.parts[1:],
    }

    if cmd == 0xb5 {
        chList, ok := m.subscriptions[*header.Uri]
        if ok {
            for _, ch := range chList {
                ch <- response
            }
        }
    } else {
        fmt.Println("send the callback", header.Uri)
    }
    return

}


func parsePart(reader io.Reader) ([]byte, error) {
    var size uint16
    binary.Read(reader, binary.BigEndian, &size)
    buf := make([]byte, size)
    _, err := io.ReadFull(reader, buf)
    return buf, err
} 

package main

import (
	"errors"
	"io"
	"strings"
)

type PlayerType byte

const (
	PlayerTypeNormal PlayerType = 0
	PlayerTypeAdmin             = 0x64
)

const (
	ProtocolVersionClassic30 = 0x07
)

const (
	PacketClientHello          = 0x00
	PacketClientSetBlock       = 0x05
	PacketClientPositionUpdate = 0x08
	PacketClientMessage        = 0x0d
)

const (
	PacketServerHello          = 0x00
	PacketServerLevelInit      = 0x02
	PacketServerLevelDataChunk = 0x03
	PacketServerLevelFinalize  = 0x04
	PacketServerSpawnPlayer    = 0x07
	PacketServerMessage        = 0x0d
	PacketServerKick           = 0x0e
)

const (
	MessageSenderServer int8 = -1
)

type ServerEncoder struct {
	w io.Writer
}

func NewServerEncoder(w io.Writer) *ServerEncoder {
	return &ServerEncoder{w}
}

func (enc *ServerEncoder) writePacket(packet []byte) error {
	n, err := enc.w.Write(packet)
	if err != nil {
		return err
	}

	if n != len(packet) {
		return errors.New("write truncated packet")
	}

	return nil
}

func (enc *ServerEncoder) WriteServerHello(name, motd string, playerType PlayerType) error {
	buf := make([]byte, 131)
	buf[0] = PacketServerHello
	buf[1] = ProtocolVersionClassic30
	err := writeString(buf[2:66], name)
	if err != nil {
		return err
	}
	err = writeString(buf[66:130], motd)
	if err != nil {
		return err
	}
	buf[130] = byte(playerType)

	return enc.writePacket(buf)
}

func (enc *ServerEncoder) WriteLevelInit() error {
	return enc.writePacket([]byte{PacketServerLevelInit})
}

func (enc *ServerEncoder) WriteLevelDataChunk(chunk []byte, sent, total int) error {
	if len(chunk) > 1024 {
		return errors.New("level data chunk too big")
	}
	if sent > total {
		return errors.New("sent > total")
	}

	buf := make([]byte, 1028)
	buf[0] = PacketServerLevelDataChunk
	writeInt16(buf[1:3], int16(len(chunk)))

	n := copy(buf[3:1027], chunk)
	for n < 1024 {
		buf[3+n] = 0x00
		n++
	}

	buf[1027] = byte(float64(sent) / float64(total) * 100)

	return enc.writePacket(buf)
}

func (enc *ServerEncoder) WriteLevelFinalize(width, depth, height int16) error {
	buf := make([]byte, 7)
	buf[0] = PacketServerLevelFinalize
	writeInt16(buf[1:3], width)
	writeInt16(buf[3:5], depth)
	writeInt16(buf[5:7], height)

	return enc.writePacket(buf)
}

func (enc *ServerEncoder) WriteSpawnPlayer(playerId int8, name string, x, y, z int16, yaw, pitch byte) error {
	buf := make([]byte, 74)
	buf[0] = PacketServerSpawnPlayer
	buf[1] = byte(playerId)
	err := writeString(buf[2:66], name)
	if err != nil {
		return err
	}
	writeInt16(buf[66:68], x)
	writeInt16(buf[68:70], y)
	writeInt16(buf[70:72], z)
	buf[72] = yaw
	buf[73] = pitch

	return enc.writePacket(buf)
}

func (enc *ServerEncoder) WriteMessage(message string, sender int8) error {
	buf := make([]byte, 66)
	buf[0] = PacketServerMessage
	buf[1] = byte(sender)
	err := writeString(buf[2:], message)
	if err != nil {
		return err
	}

	return enc.writePacket(buf)
}

func (enc *ServerEncoder) WriteKick(reason string) error {
	buf := make([]byte, 65)
	buf[0] = PacketServerKick
	err := writeString(buf[1:], reason)
	if err != nil {
		return err
	}

	return enc.writePacket(buf)
}

type ClientDecoder struct {
	r io.Reader
}

func NewClientDecoder(r io.Reader) *ClientDecoder {
	return &ClientDecoder{r}
}

func (dec *ClientDecoder) readBuf(buf []byte) error {
	n, err := dec.r.Read(buf)

	if err != nil {
		return err
	} else if n != len(buf) {
		return errors.New("truncated read")
	}

	return nil
}

func (dec *ClientDecoder) readByte() (byte, error) {
	buf := make([]byte, 1)

	if err := dec.readBuf(buf); err != nil {
		return 0, err
	}

	return buf[0], nil
}

func (dec *ClientDecoder) readInt16() (i int16, err error) {
	buf := make([]byte, 2)

	if err := dec.readBuf(buf); err != nil {
		return 0, err
	}

	return int16(buf[0])<<8 | int16(buf[1]), nil
}

func (dec *ClientDecoder) readString() (string, error) {
	buf := make([]byte, 64)

	if err := dec.readBuf(buf); err != nil {
		return "", err
	}

	return strings.TrimRight(string(buf), " "), nil
}

func (dec *ClientDecoder) NextPacketID() (byte, error) {
	id, err := dec.readByte()
	if err != nil {
		return 0, err
	}
	if id != PacketClientHello && id != PacketClientSetBlock && id != PacketClientPositionUpdate && id != PacketClientMessage {
		return 0, errors.New("client sent invalid packet ID")
	}

	return id, nil
}

func (dec *ClientDecoder) ReadClientHello() (name, mppass string, err error) {
	protocolVersion, err := dec.readByte()
	if err != nil {
		return
	} else if protocolVersion != ProtocolVersionClassic30 {
		err = errors.New("invalid protocol version")
		return
	}

	if name, err = dec.readString(); err != nil {
		return
	}
	if mppass, err = dec.readString(); err != nil {
		return
	}
	// Unused byte
	if _, err = dec.readByte(); err != nil {
		return
	}

	return name, mppass, err
}

func (dec *ClientDecoder) ReadSetBlock() (x, y, z int16, mode, blockType byte, err error) {
	x, err = dec.readInt16()
	if err != nil {
		return
	}
	y, err = dec.readInt16()
	if err != nil {
		return
	}
	z, err = dec.readInt16()
	if err != nil {
		return
	}
	mode, err = dec.readByte()
	if err != nil {
		return
	}
	blockType, err = dec.readByte()

	return
}

func (dec *ClientDecoder) ReadPositionUpdate() (x, y, z int16, yaw, pitch byte, err error) {
	// PlayerID is always 255
	if _, err = dec.readByte(); err != nil {
		return
	}
	x, err = dec.readInt16()
	if err != nil {
		return
	}
	y, err = dec.readInt16()
	if err != nil {
		return
	}
	z, err = dec.readInt16()
	if err != nil {
		return
	}
	yaw, err = dec.readByte()
	if err != nil {
		return
	}
	pitch, err = dec.readByte()

	return
}

func (dec *ClientDecoder) ReadMessage() (message string, err error) {
	// First byte is an unused ID
	if _, err = dec.readByte(); err != nil {
		return
	}
	message, err = dec.readString()

	return
}

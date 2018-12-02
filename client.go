package main

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"strings"
)

var ErrInvalidMessage = errors.New("invalid message")

type Client struct {
	conn    net.Conn
	encoder *ServerEncoder
	decoder *ClientDecoder
	museum  *Museum

	name           string
	warnedSetBlock bool
}

func (c *Client) MainLoop() {
	defer func() {
		c.log("Closing connection")
		c.conn.Close()
	}()

	err := c.handshake()
	if err != nil {
		c.log("[ERROR] handshake failed: %s", err.Error())
		return
	}

	level, err := c.museum.GetDefaultLevel()
	if err != nil {
		log.Printf("[ERROR] failed to load default level: %s", err.Error())
		c.Kick("Failed to load level")
		return
	}

	err = c.SendLevel(level)
	if err != nil {
		c.log("[ERROR] Initial SendLevel failed: %s", err.Error())
		return
	}

	c.about()

	for {
		packetId, err := c.decoder.NextPacketID()
		if err == io.EOF {
			// Close quietly on EOF
			return
		} else if err != nil {
			c.log("[ERROR] read failed: %s", err.Error())
			return
		}

		switch packetId {
		case PacketClientHello:
			_, _, err = c.decoder.ReadClientHello()
		case PacketClientSetBlock:
			_, _, _, _, _, err = c.decoder.ReadSetBlock()
			if !c.warnedSetBlock {
				c.SendMessage("This server is a view-only archive of old levels.  Your changes won't be saved", MessageSenderServer)
				c.warnedSetBlock = true
			}
		case PacketClientPositionUpdate:
			_, _, _, _, _, err = c.decoder.ReadPositionUpdate()
		case PacketClientMessage:
			message, err := c.decoder.ReadMessage()
			if err == nil {
				if message[0] == '/' {
					c.handleCommand(message)
				} else {
					c.SendMessage("Chat is disabled for this server", MessageSenderServer)
				}
			}
		default:
			err = errors.New("Unhandled packet ID")
		}

		if err != nil {
			c.log("[ERROR] failed to decode client packet: %s", err.Error())
			return
		}
	}
}

func (c *Client) handshake() error {
	packetId, err := c.decoder.NextPacketID()
	if err != nil {
		return err
	} else if packetId != PacketClientHello {
		return errors.New("expected ClientHello")
	}

	name, _, err := c.decoder.ReadClientHello()
	if err != nil {
		return err
	} else {
		c.log("Logged in as %s", name)
		c.name = name
	}

	return c.encoder.WriteServerHello(
		c.museum.Name,
		c.museum.MOTD,
		PlayerTypeAdmin)
}

func (c *Client) SendLevel(level LevelDescriptor) error {
	lvl, err := ReadLevel(level.Path)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	gzout := gzip.NewWriter(buf)
	defer gzout.Close()

	if err = writeInt32(gzout, len(lvl.Blocks)); err != nil {
		return err
	}

	if _, err = io.Copy(gzout, bytes.NewReader(lvl.Blocks)); err != nil {
		return err
	}

	if err = gzout.Flush(); err != nil {
		return err
	}

	mapb := buf.Bytes()

	if err = c.encoder.WriteLevelInit(); err != nil {
		return err
	}

	for i := 0; i < len(mapb); i += 1024 {
		length := len(mapb) - i
		if length > 1024 {
			length = 1024
		}

		if err := c.encoder.WriteLevelDataChunk(mapb[i:i+length], i, len(mapb)); err != nil {
			return err
		}
	}

	if err = c.encoder.WriteLevelFinalize(lvl.Width, lvl.Depth, lvl.Height); err != nil {
		return err
	}

	if err = c.encoder.WriteSpawnPlayer(-1, c.name, lvl.Spawn.X, lvl.Spawn.Y, lvl.Spawn.Z, 0, 0); err != nil {
		return err
	}

	c.SendMessage(fmt.Sprintf(
		"This level is &c%s&e, from %s",
		level.Name,
		level.Datestring),
		MessageSenderServer)
	return nil
}

func (c *Client) SendMessage(message string, sender int8) {
	mbytes := []byte(message)
	if mbytes[len(mbytes)-1] == '&' {
		log.Printf("[ERROR] Cannot send message '%s'", message)
		return
	}

	for len(mbytes) > 64 {
		line := mbytes[:64]
		if x := bytes.LastIndexByte(line, ' '); x > 0 {
			line = line[:x]
		} else if line[len(line)-1] == '&' {
			// Sending an incomplete color-code will crash the game
			line = line[:len(line)-1]
		}

		err := c.encoder.WriteMessage(string(line), sender)
		if err != nil {
			c.log("[ERROR] Message send failed: %s", err.Error())
			return
		}

		mbytes = append([]byte("> "), mbytes[len(line):]...)
	}

	if string(mbytes) != "> " {
		if err := c.encoder.WriteMessage(string(mbytes), sender); err != nil {
			log.Printf("[ERROR] Message send failed: %s", err.Error())
		}
	}
}

func (c *Client) Kick(reason string) error {
	return c.encoder.WriteKick(reason)
}

func (c *Client) handleCommand(message string) {
	args := strings.Split(message, " ")

	switch args[0] {
	case "/help":
		c.SendMessage("Available commands:", MessageSenderServer)
		c.SendMessage("- &c/about&e: show information about this server", MessageSenderServer)
		c.SendMessage("- &c/levels&e: list available levels", MessageSenderServer)
		c.SendMessage("- &c/goto <levelname>&e: warp to another level", MessageSenderServer)
		c.SendMessage("- &c/random&e: warp to a random level", MessageSenderServer)
	case "/about":
		c.about()
	case "/levels":
		levels := c.museum.ListLevelNames()
		c.SendMessage("Available levels: "+strings.Join(levels, ", "), MessageSenderServer)
	case "/goto":
		if len(args) < 2 {
			c.SendMessage("Usage: &c/goto <levelname>", MessageSenderServer)
			return
		}
		levelname := args[1]
		if level, err := c.museum.GetLevel(levelname); err != nil {
			c.SendMessage("Unknown level &c"+levelname, MessageSenderServer)
		} else {
			if err = c.SendLevel(level); err != nil {
				c.log("[ERROR] Failed to send level: %s", err.Error())
			} else {
				c.log("Visiting level %s", level.Name)
			}
		}
	case "/random":
		levels := c.museum.ListLevelNames()
		index := rand.Intn(len(levels))
		level, err := c.museum.GetLevel(levels[index])
		if err != nil {
			// Should not happen since this level was just returned by ListLevels()
			log.Printf("[ERROR] /random failed to retrieve level %s: %s", levels[index], err.Error())
			c.SendMessage("Internal error locating level.  Try again later", MessageSenderServer)
			return
		}

		if err = c.SendLevel(level); err != nil {
			c.log("[ERROR] Failed to send level: %s", err.Error())
		} else {
			c.log("Visiting level %s", level.Name)
		}
	default:
		c.SendMessage("Unknown command &c"+args[0], MessageSenderServer)
	}
}

func (c *Client) about() {
	c.SendMessage("Welcome to &c"+c.museum.Name, MessageSenderServer)
	c.SendMessage("This server is a view-only archive of Minecraft levels circa 2009-2010", MessageSenderServer)
	c.SendMessage("For information about available commands, type &c/help", MessageSenderServer)
	c.SendMessage("For questions or comments, contact &ccalzoneman&e on &circ.esper.net", MessageSenderServer)
}

func (c *Client) log(message string, args ...interface{}) {
	args = append([]interface{}{c.conn.RemoteAddr()}, args...)
	log.Printf("<%s> "+message, args...)
}

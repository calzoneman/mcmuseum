package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"
)

var (
	ServerName      = flag.String("name", "", "Server name")
	ServerMOTD      = flag.String("motd", "", "Server MOTD")
	ManifestFile    = flag.String("manifest", "manifest.csv", "Level manifest file")
	Port            = flag.Int("port", 25565, "Port to listen on")
	ConnectionLimit = flag.Int("maxconns", 32, "Maximum number of connected players")
	SendHeartbeat   = flag.Bool("heartbeat", false, "Send heartbeats to classicube.net")
	Public          = flag.Bool("public", false, "List the server publicly on classicube.net")
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.LUTC | log.Lshortfile)

	flag.Parse()

	if *ServerName == "" {
		log.Fatalf("-name is required")
		return
	}
	if *Public && !*SendHeartbeat {
		log.Fatalf("-public is only permitted if -heartbeat is set")
	}

	rand.Seed(time.Now().Unix())
	museum, err := NewMuseum(
		*ServerName,
		*ServerMOTD,
		*ManifestFile)
	if err != nil {
		log.Fatalf("Failed to load %s: %s", *ManifestFile, err.Error())
	}

	connects := make(chan bool)
	disconnects := make(chan bool)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *Port))
	if err != nil {
		log.Fatalf("Failed to start listener: %s", err.Error())
	}
	log.Printf("Listening on :%d", *Port)

	if *SendHeartbeat {
		log.Printf("Sending initial heartbeat")

		hb := &Heartbeat{
			Name:            *ServerName,
			Port:            *Port,
			NumConnected:    0,
			ConnectionLimit: *ConnectionLimit,
			Public:          *Public,
			Salt:            "PJSalt", // Salt not really needed since this server doesn't verify names
		}

		playURL, err := hb.Send()
		if err != nil {
			log.Fatalf("Failed to send initial heartbeat: %s", err.Error())
		}

		log.Printf("Play URL = %s", playURL)
		go startHeartbeats(hb, connects, disconnects)
	} else {
		go func() {
			for {
				select {
				case <-connects:
				case <-disconnects:
				}
			}
		}()
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatalf("Failed to accept connection: %s", err.Error())
		}

		log.Printf("Accepted connection from %s", conn.RemoteAddr())
		connects <- true
		client := &Client{
			conn:    conn,
			encoder: NewServerEncoder(conn),
			decoder: NewClientDecoder(conn),
			museum:  museum,
		}

		go func() {
			client.MainLoop()
			disconnects <- true
		}()
	}
}

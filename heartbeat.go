package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
)

const ClassicubeEndpoint = "https://www.classicube.net/server/heartbeat/"

type Heartbeat struct {
	Name            string
	Port            int
	NumConnected    int
	ConnectionLimit int
	Public          bool
	Salt            string
	Software        string
}

func (hb *Heartbeat) Send() (string, error) {
	query := fmt.Sprintf(
		"?name=%s&port=%d&users=%d&max=%d&public=%v&salt=%s",
		url.QueryEscape(hb.Name),
		hb.Port,
		hb.NumConnected,
		hb.ConnectionLimit,
		hb.Public,
		url.QueryEscape(hb.Salt))
	if hb.Software != "" {
		query += "&software=" + url.QueryEscape(hb.Software)
	}

	res, err := http.Get(ClassicubeEndpoint + query)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Printf("[ERROR] failed to read response from failed heartbeat: %s", err.Error())
		} else {
			log.Printf("[ERROR] failed to send heartbeat: %s", body)
		}
		return "", errors.New("Classicube returned HTTP " + res.Status)
	}

	playURL, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", errors.New("Error retreiving play URL from heartbeat")
	}

	return string(playURL), nil
}

func startHeartbeats(hb *Heartbeat, connects <-chan bool, disconnects <-chan bool) {
	ticker := time.NewTicker(time.Minute)

	for {
		select {
		case <-connects:
			hb.NumConnected++
		case <-disconnects:
			hb.NumConnected--
		case <-ticker.C:
			log.Println("Sending heartbeat")
			_, err := hb.Send()
			if err != nil {
				log.Printf("Heartbeat failed: %s", err.Error())
			}
		}
	}
}

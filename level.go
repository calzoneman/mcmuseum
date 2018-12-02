package main

import (
	"compress/gzip"
	"errors"
	"io"
	"log"
	"os"
)

type Spawnpoint struct {
	X    int16
	Y    int16
	Z    int16
	RotX byte
	RotY byte
}

type Level struct {
	Blocks []byte
	Width  int16
	Depth  int16
	Height int16
	Spawn  Spawnpoint
}

func ReadLevel(filename string) (*Level, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	gzin, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer gzin.Close()

	width, err := readInt16(gzin)
	if err != nil {
		return nil, err
	}
	depth, err := readInt16(gzin)
	if err != nil {
		return nil, err
	}
	height, err := readInt16(gzin)
	if err != nil {
		return nil, err
	}

	spawn := Spawnpoint{}
	spawn.X, err = readInt16(gzin)
	if err != nil {
		return nil, err
	}
	spawn.Y, err = readInt16(gzin)
	if err != nil {
		return nil, err
	}
	spawn.Z, err = readInt16(gzin)
	if err != nil {
		return nil, err
	}

	blocks := make([]byte, int(width)*int(depth)*int(height))
	i := 0
	for i < len(blocks) {
		n, err := gzin.Read(blocks[i:])
		if err != nil && err != io.EOF {
			return nil, err
		}

		i += n
	}

	if i < len(blocks) {
		return nil, errors.New("ReadLevel: file too short")
	}

	log.Printf(
		"Loaded level from %s (size = %d x %d x %d)",
		filename,
		width,
		depth,
		height)

	return &Level{
		Blocks: blocks,
		Width:  width,
		Depth:  depth,
		Height: height,
		Spawn:  spawn,
	}, nil
}

package main

import (
	"encoding/csv"
	"errors"
	"os"
)

var ErrLevelNotFound = errors.New("level not found")

type LevelDescriptor struct {
	Name       string
	Path       string
	Datestring string
}

type Museum struct {
	Name   string
	MOTD   string
	levels []LevelDescriptor
}

func NewMuseum(name, motd, manifestFilename string) (*Museum, error) {
	file, err := os.Open(manifestFilename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)

	lines, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	levels := []LevelDescriptor{}
	for _, line := range lines {
		if len(line) != 3 {
			return nil, errors.New("manifest file is corrupt")
		}

		levels = append(levels, LevelDescriptor{
			Name:       line[0],
			Path:       line[1],
			Datestring: line[2],
		})
	}

	return &Museum{
		Name:   name,
		MOTD:   motd,
		levels: levels,
	}, nil
}

func (m *Museum) ListLevelNames() []string {
	names := []string{}

	for _, level := range m.levels {
		names = append(names, level.Name)
	}

	return names
}

func (m *Museum) GetLevel(name string) (LevelDescriptor, error) {
	for _, level := range m.levels {
		if level.Name == name {
			return level, nil
		}
	}

	return LevelDescriptor{}, ErrLevelNotFound
}

func (m *Museum) GetDefaultLevel() (LevelDescriptor, error) {
	for _, level := range m.levels {
		return level, nil
	}

	return LevelDescriptor{}, ErrLevelNotFound
}

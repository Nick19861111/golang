package discovery

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type Server struct {
	Name    string `json:"name"`
	Addr    string `json:"addr"`
	Version string `json:"version"`
	Weight  int    `json:"weight"`
	Ttl     int64  `json:"ttl"`
}

func (s Server) BuildRegisterKey() string {
	if s.Version == "" {
		return fmt.Sprintf("/%s/%s", s.Name, s.Addr)
	}
	return fmt.Sprintf("/%s/%s/%s", s.Name, s.Version, s.Addr)
}

func ParseValue(val []byte) (Server, error) {
	server := Server{}
	if err := json.Unmarshal(val, &server); err != nil {
		return server, err
	}

	return server, nil
}

func ParseKey(key string) (Server, error) {
	strs := strings.Split(key, "/")
	if len(strs) == 2 {
		//no version
		return Server{
			Name: strs[0],
			Addr: strs[1],
		}, nil
	}
	if len(strs) == 3 {
		//has version
		return Server{
			Name:    strs[0],
			Version: strs[1],
			Addr:    strs[2],
		}, nil
	}
	return Server{}, errors.New("invalid key")
}

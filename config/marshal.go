package config

import (
	"errors"
	"io/ioutil"
	"os"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func MarshalServersToFile(fn string, servers map[int64]*Server) error {
	sp := &Servers{
		Servers: servers,
	}
	return Marshal(fn, sp)
}

func UnmarshalServersFromFile(fn string) (map[int64]*Server, error) {
	sp := &Servers{}
	err := Unmarshal(fn, sp)
	return sp.Servers, err
}

func MarshalGroupsToFile(fn string, groups map[int64]*Group) error {
	sp := &Groups{
		Groups: groups,
	}
	return Marshal(fn, sp)
}

func UnmarshalGroupsFromFile(fn string) (map[int64]*Group, error) {
	sp := &Groups{}
	err := Unmarshal(fn, sp)
	return sp.Groups, err
}

func Marshal(fn string, m protoreflect.ProtoMessage) error {
	file, err := os.Create(fn)
	if err != nil {
		return err
	}
	b, err := protojson.Marshal(m)
	if err != nil {
		return err
	}
	_, err = file.Write(b)
	return err
}

func Unmarshal(fn string, m protoreflect.ProtoMessage) error {
	file, err := os.Open(fn)
	if err != nil {
		return err
	}
	b, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}
	return protojson.Unmarshal(b, m)
}

func UnmarshalNetwork(fn string, addr string) (int, *Server, map[string]int, map[int][]byte, error) {
	servers, err := UnmarshalServersFromFile(fn)
	if err != nil {
		return 0, nil, nil, nil, err
	}
	peers := make(map[string]int)
	verificationKeys := make(map[int][]byte)
	for sid, s := range servers {
		peers[s.Address] = int(sid)
		verificationKeys[int(sid)] = s.VerificationKey
	}
	for sid, s := range servers {
		if s.Address == addr {
			return int(sid), s, peers, verificationKeys, nil
		}
	}
	return 0, nil, nil, nil, errors.New("Address not found")
}

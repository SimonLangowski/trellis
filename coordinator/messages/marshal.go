package coord

import (
	"io/ioutil"
	"os"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func MarshalKeysToFile(fn string, keys []*BootstrapKey) error {
	sp := &PathKeys{
		Keys: keys,
	}
	return Marshal(fn, sp)
}

func UnmarshalKeysFromFile(fn string) ([]*BootstrapKey, error) {
	sp := &PathKeys{}
	err := Unmarshal(fn, sp)
	return sp.Keys, err
}

func MarshalMessagesToFile(fn string, servers []int64, messages [][]byte) error {
	sp := &TestMessages{
		StartingServers: servers,
		Ciphers:         messages,
	}
	return Marshal(fn, sp)
}

func UnmarshalMessagesFromFile(fn string) ([]int64, [][]byte, error) {
	sp := &TestMessages{}
	err := Unmarshal(fn, sp)
	return sp.StartingServers, sp.Ciphers, err
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

func (i *RoundInfo) Copy() *RoundInfo {
	return &RoundInfo{
		Round:             i.Round,
		NumLayers:         i.NumLayers,
		BinSize:           i.BinSize,
		PathEstablishment: i.PathEstablishment,
		LastLayer:         i.LastLayer,
		MessageSize:       i.MessageSize,
		StartId:           i.StartId,
		EndId:             i.EndId,
		PublicKeys:        i.PublicKeys,
		ReceiptLayer:      i.ReceiptLayer,
		BoomerangLimit:    i.BoomerangLimit,
		NextLayer:         i.NextLayer,
		Check:             i.Check,
		Interval:          i.Interval,
		SkipPathGen:       i.SkipPathGen,
	}
}

package coordinator

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/simonlangowski/lightning1/config"
	coord "github.com/simonlangowski/lightning1/coordinator/messages"
	"github.com/simonlangowski/lightning1/crypto"
	"github.com/simonlangowski/lightning1/crypto/pairing/mcl"
	"github.com/simonlangowski/lightning1/crypto/token"
	"github.com/simonlangowski/lightning1/errors"
	"github.com/simonlangowski/lightning1/server/prepareMessages"
)

// The coordinator simulates the glocal clock time when the round begins, the time when receipts should have been received by, etc.
// It also allows the experimenter to set parameters and measure the time things take

type Coordinator struct {
	privateKeys     []map[int64]*coord.KeyInformation
	groupSecretKeys groupSecretKeys
	publicKeys      *coord.KeyInformation
	Net             *CoordinatorNetwork
	mu              sync.Mutex
}

type groupSecretKeys struct {
	TokenSecretKey []byte
	Ssk            crypto.DHPrivateKey
	GroupKey       []byte
}

type Experiment struct {
	Info                     *coord.RoundInfo
	NumMessages              int
	Profile                  string
	KeyGen                   bool
	DoRound                  bool
	LoadKeys                 bool
	Passed                   bool
	KeyGenTime               time.Duration
	SetupTime                time.Duration
	ClientAndServerTokenTime time.Duration
	ServerRoundTime          time.Duration
	ExperimentStartTime      time.Time
	Notes                    interface{}
}

func NewCoordinator(net *CoordinatorNetwork) *Coordinator {
	cfgs := net.ServerConfigs
	groups := net.GroupConfigs
	c := &Coordinator{
		privateKeys: make([]map[int64]*coord.KeyInformation, len(cfgs)),
		publicKeys:  &coord.KeyInformation{},
		Net:         net,
	}
	for i := range c.privateKeys {
		c.privateKeys[i] = make(map[int64]*coord.KeyInformation)
	}
	for gid, group := range groups {
		for _, sid := range group.Servers {
			c.privateKeys[sid][gid] = &coord.KeyInformation{}
		}
	}
	return c
}

func (c *Coordinator) NewExperiment(round, numLayers, numServers, numMessages int, notes interface{}) *Experiment {
	return &Experiment{
		Info: &coord.RoundInfo{
			Round:             int64(round),
			NumLayers:         int64(numLayers),
			BinSize:           int64(config.BinSize2(numLayers, numServers, numMessages, -32)),
			PathEstablishment: true,
			LastLayer:         false,
			MessageSize:       8,
			StartId:           0,
			EndId:             int64(numMessages),
			Check:             true,
		},
		NumMessages: numMessages,
		DoRound:     true,
		Notes:       notes,
	}
}

func (c *Coordinator) DoAction(exp *Experiment) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(exp.Profile) > 0 {
		f, err := os.Create(exp.Profile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	exp.ExperimentStartTime = time.Now()

	if exp.KeyGen {
		c.KeyGenToken()
		c.GenDHKeys()
	}
	if exp.KeyGen || exp.LoadKeys {
		err := c.Net.SendKeys(c.privateKeys, c.publicKeys)
		if err != nil {
			log.Printf("Key gen")
			return err
		}
	}
	keyGenTime := time.Now()
	if exp.DoRound {
		if !exp.Info.PathEstablishment || exp.Info.Round == 0 {
			err := c.Net.SendRoundSetup(exp.Info)
			if err != nil {
				log.Printf("Round setup")
				return err
			}
			setupTime := time.Now()
			if exp.LoadKeys {
				c.Net.clients.SendLoadedMessages(exp.Info)
			} else {
				err = c.Net.SendClientStart(exp.Info, exp.NumMessages)
				if err != nil {
					log.Printf("Client start")
					return err
				}
			}
			exp.SetupTime = setupTime.Sub(keyGenTime)
			clientAndServerTokenTime := time.Now()
			exp.ClientAndServerTokenTime = clientAndServerTokenTime.Sub(setupTime)
		}
		roundStartTime := time.Now()
		if !(exp.Info.PathEstablishment && exp.Info.Round == 0) {
			err := c.Net.SendRoundStart(exp.Info)
			if err != nil {
				log.Printf("Server start")
				return err
			}
		}
		if exp.Info.PathEstablishment {
			if exp.Info.ReceiptLayer == 0 && exp.Info.Round != 0 {
				err := c.Net.CheckClientReceipt(exp.Info, exp.NumMessages)
				if err != nil {
					log.Printf("Client receipts")
					return err
				}
				exp.Passed = true
			} else if exp.Info.ReceiptLayer > 0 {
				// these receipts are only checked for test purposes
				// they could not be checked without breaking anonymity
				messages, err := c.Net.GetMessages(exp.Info)
				if err != nil {
					log.Printf("Get messages")
					return err
				}
				if c.Net.clientNetType == inprocess && exp.Info.Check {
					exp.Passed = c.CheckReceipts(messages, int(exp.Info.Round), c.Net.clients.Clients)
					if !exp.Passed {
						log.Printf("Client receipts in process")
						return errors.WrongReceipt()
					}
				} else {
					// skip check - however getMessages ensures the round has completed which is still important
					exp.Passed = true
				}
			} else {
				exp.Passed = true
			}
		} else {
			messages, err := c.Net.GetMessages(exp.Info)
			if err != nil {
				log.Printf("Get messages")
				return err
			}
			exp.Passed = c.Check(messages, exp.NumMessages)
		}
		endTime := time.Now()
		exp.ServerRoundTime = endTime.Sub(roundStartTime)
	}

	exp.KeyGenTime = keyGenTime.Sub(exp.ExperimentStartTime)
	return nil
}

func (c *Coordinator) Check(messages [][]byte, numExpected int) bool {
	seen := make(map[uint64]bool)
	// test messages are consecutive integers up to numExpected
	for _, m := range messages {
		c := binary.LittleEndian.Uint64(m)
		if c < uint64(numExpected) {
			seen[c] = true
		} else {
			return false
		}
	}
	return len(seen) == numExpected
}

func (c *Coordinator) CheckReceipts(receipts [][]byte, round int, clients map[int64]*prepareMessages.Client) bool {
	// receipts are not sorted
	if len(receipts) != len(clients) {
		log.Printf("Error: number of receipts and number of clients mismatched")
		return false
	}
	ok := true
	for _, c := range clients {
		receipt := c.Receipts[round]
		found := false
		for _, r := range receipts {
			if bytes.Equal(receipt, r) {
				found = true
				break
			}
		}
		if !found {
			log.Printf("Could not find receipt %v", receipt)
			ok = false
		}
	}
	return ok
}

// return keys to send to each server
func (c *Coordinator) KeyGenToken() {
	tokenSecretKey := mcl.Fr{}
	tokenSecretKey.Random()
	c.keyGenToken(&tokenSecretKey)
	c.groupSecretKeys.TokenSecretKey = tokenSecretKey.Serialize()
}

func (c *Coordinator) keyGenToken(tokenSecretKey *mcl.Fr) {
	if config.SkipToken {
		log.Print("Warning: Using fixed token key is insecure")
		tokenSecretKey = &token.SecretKey.Share
	}
	for gid, group := range c.Net.GroupConfigs {
		shares, pk, _ := token.MockKeyGen(len(group.Servers), tokenSecretKey)
		for i, sid := range group.Servers {
			k := c.privateKeys[sid][gid]
			k.GroupId = gid
			k.TokenPublicKey = make([]byte, pk.Len())
			pk.PackTo(k.TokenPublicKey)
			k.TokenKeyShare = make([]byte, shares[i].Share.Len())
			shares[i].Share.PackTo(k.TokenKeyShare)
		}
	}
	tokenPublicKey := token.TokenPublicKey{}
	tokenPublicKey.X = token.NewTokenSigningKey(tokenSecretKey).X
	c.publicKeys.TokenPublicKey = make([]byte, tokenPublicKey.Len())
	tokenPublicKey.PackTo(c.publicKeys.TokenPublicKey)
}

func (c *Coordinator) GenDHKeys() {
	ssk, spk := crypto.NewDHKeyPair()
	groupKey := make([]byte, spk.Len())
	spk.PackTo(groupKey)
	c.groupSecretKeys.Ssk = ssk
	c.groupSecretKeys.GroupKey = groupKey
	c.genDHKeys(ssk, groupKey)
}

func (c *Coordinator) genDHKeys(ssk crypto.DHPrivateKey, groupKey []byte) {
	for gid, group := range c.Net.GroupConfigs {
		shares := crypto.AdditiveShares(&ssk, len(group.Servers))
		for idx, sid := range group.Servers {
			b := shares[idx].Bytes()
			k := c.privateKeys[sid][gid]
			k.GroupId = gid
			k.GroupShare = b
			k.GroupKey = groupKey
		}
	}
	c.publicKeys.GroupKey = groupKey
}

func (e *Experiment) RecordToFile(fn string) {
	e.Info.PublicKeys = nil
	data, err := json.MarshalIndent(e, "", " ")
	if err != nil {
		panic(err)
	}
	f, err := os.OpenFile(fn, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	_, err = f.Write(data)
	if err != nil {
		panic(err)
	}
}

func (c *Coordinator) WriteKeys(fn string) {
	// note that server keys are in the server config file
	// these are the keys for the anytrust groups
	// write group secret token key and dh key
	c.KeyGenToken()
	c.GenDHKeys()
	err := c.Net.SendKeys(c.privateKeys, c.publicKeys)
	if err != nil {
		panic(err)
	}

	f, err := os.Create(fn)
	if err != nil {
		panic(err)
	}
	b, err := json.Marshal(c.groupSecretKeys)
	if err != nil {
		panic(err)
	}
	_, err = f.Write(b)
	if err != nil {
		panic(err)
	}
}

func (c *Coordinator) LoadKeys(fn string) {
	b, err := ioutil.ReadFile(fn)
	if err != nil {
		panic(err)
	}
	// read group secret token key and dh key
	err = json.Unmarshal(b, &c.groupSecretKeys)
	if err != nil {
		panic(err)
	}
	// regenerate shares for the current group configuration
	tokenSecretKey := mcl.Fr{}
	tokenSecretKey.Deserialize(c.groupSecretKeys.TokenSecretKey)
	c.keyGenToken(&tokenSecretKey)
	c.genDHKeys(c.groupSecretKeys.Ssk, c.groupSecretKeys.GroupKey)
}

// record the messages to the designated file instead of submitting to the servers
func (c *Coordinator) WriteMessages(fn string, exp *Experiment) {
	if c.Net.clientNetType != inprocess {
		panic(errors.UnimplementedError())
	}
	c.Net.clients.RecordMessageFile = fn
	err := c.Net.SendRoundSetup(exp.Info)
	if err != nil {
		panic(err)
	}
	err = c.Net.SendClientStart(exp.Info, exp.NumMessages)
	if err != nil {
		panic(err)
	}
}

func (c *Coordinator) LoadMessages(fn string) {
	if c.Net.clientNetType != inprocess {
		panic(errors.UnimplementedError())
	}
	c.Net.clients.RecordMessageFile = fn
}

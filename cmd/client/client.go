package main

import (
	"log"
	"os"

	"github.com/simonlangowski/lightning1/client"
	"github.com/simonlangowski/lightning1/config"
	"github.com/simonlangowski/lightning1/errors"
	"github.com/simonlangowski/lightning1/network"
)

func main() {
	serversFile := os.Args[1]
	groupsFile := os.Args[2]
	clientsFile := os.Args[3]
	addr := os.Args[4]
	errors.Addr = addr
	servers, err := config.UnmarshalServersFromFile(serversFile)
	if err != nil {
		log.Fatalf("Could not read servers file %s", serversFile)
	}
	groups, err := config.UnmarshalGroupsFromFile(groupsFile)
	if err != nil {
		log.Fatalf("Could not read group file %s", groupsFile)
	}
	clients, err := config.UnmarshalServersFromFile(clientsFile)
	if err != nil {
		log.Fatalf("Could not read clients file %s", clientsFile)
	}

	clientRunner := client.NewClientRunner(servers, groups)
	err = clientRunner.Connect()
	if err != nil {
		log.Fatalf("Could not make clients %v", err)
	}
	network.RunServer(nil, clientRunner, clients, addr)
}

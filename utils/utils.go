package utils

import (
	"flag"
	"strings"
)

//GetGossiperInfos parses from the command line to return gossiper's Name, UIPort, IP and listening port (move to Gossiper?)
func GetGossiperInfos() (string, string, string, string, []string) {
	gossiperName := flag.String("name", "my_gossiper", "Gossiper's name")
	clientPort := flag.String("UIPort", "8080", "Port to listen for the client")
	gossipersAddr := flag.String("gossipAddr", "127.0.0.1:5000", "Gossiper's IP and port")
	peersList := flag.String("peers", "", "List of peers' IP address and port")

	flag.Parse()

	gossiperAddr := strings.Split(*gossipersAddr, ":")
	peersAddr := strings.Split(*peersList, ",")
	gossiperIP := gossiperAddr[0]
	gossiperPort := gossiperAddr[1]

	return *gossiperName, *clientPort, gossiperIP, gossiperPort, peersAddr
}

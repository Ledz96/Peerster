package main

import (
	"github.com/AlessandroBianchi/Peerster/gossiper"
)

func main() {
	g := gossiper.New()

	g.SetInfos()

	go g.ListenForClients()
	g.ListenForPeers()
}

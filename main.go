package main

import (
	"fmt"

	"github.com/AlessandroBianchi/Peerster/gossiper"
)

func main() {
	g := gossiper.New()

	g.SetInfos()

	fmt.Println(g)

	go g.ListenForClients()
	g.ListenForPeers()
}

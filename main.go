package main

import (
	"github.com/AlessandroBianchi/Peerster/frontend"
	"github.com/AlessandroBianchi/Peerster/gossiper"
)

func main() {
	g := gossiper.New()

	g.SetInfos()
	s := frontend.New(g)

	if g.Simple() {
		go g.SimpleHandleClientMessages()
		g.SimpleHandlePeersMessages()
	} else {
		go g.HandleClientMessages()
		if g.IsAntiEntropyActive() {
			go g.AntiEntropy()
		}
		go s.Start()
		g.HandlePeersMessages()
	}
}

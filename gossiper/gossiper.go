package gossiper

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"sort"
	"strings"
	"time"

	gossippacket "github.com/AlessandroBianchi/Peerster/gossippacket"
	"github.com/AlessandroBianchi/Peerster/message"
	"github.com/dedis/protobuf"
)

const buffsize int = 1024

var messageID uint32
var timerLength time.Duration

type status int

const (
	have = iota
	want
	equal
)

func init() {
	rand.Seed(time.Now().UnixNano())
	messageID = 0
	timerLength = 10
}

type gossiper struct {
	name       string
	udpAddr    *net.UDPAddr
	udpConn    *net.UDPConn
	clientAddr *net.UDPAddr
	clientConn *net.UDPConn
	peers      map[string](*net.UDPConn)
	simpleMode bool
	myStatus   map[string]uint32
	rumorMsgs  map[string][]*message.RumorMessage
	channels   map[string][]chan *gossippacket.GossipPacket
}

//New creates a new gossiper
func New() *gossiper {
	g := gossiper{}
	g.peers = make(map[string](*net.UDPConn))
	g.myStatus = make(map[string]uint32)
	g.rumorMsgs = make(map[string][]*message.RumorMessage)
	g.channels = make(map[string][]chan *gossippacket.GossipPacket)
	return &g
}

//Simple returns the simple bool variable, stating whether or not the program runs in simple mode
func (g gossiper) Simple() bool {
	return g.simpleMode
}

//SetInfos read infos from command line and stores them in the structure
func (g *gossiper) SetInfos() {
	var peersAddr []*net.UDPAddr
	g.name, g.udpAddr, g.clientAddr, peersAddr, g.simpleMode = getInfosFromCL()
	for _, peerAddr := range peersAddr {
		peerConn, err := net.DialUDP("udp4", nil, peerAddr)
		if err != nil {
			fmt.Println("Error: could not connect to given peer")
			os.Exit(-1)
		}
		g.peers[peerAddr.String()] = peerConn
	}
}

//AddPeer adds a peer to the gossiper
func (g *gossiper) AddPeer(peerAddr *(net.UDPAddr)) {
	peerConn, err := net.DialUDP("udp4", nil, peerAddr)
	if err != nil {
		fmt.Println("Error: could not connect to given peer")
		os.Exit(-1)
	}
	g.peers[peerAddr.String()] = peerConn
}

func (g *gossiper) ConnectToClient() {
	fmt.Println("Attempting client connection")

	clientConn, err := net.ListenUDP("udp4", g.clientAddr)
	if err != nil {
		fmt.Printf("Error: can't listen for clients on socket; %v\n", err)
		os.Exit(-1)
	}
	g.clientConn = clientConn

	fmt.Printf("Listening for client on port %d\n", g.clientAddr.Port)
}

func (g *gossiper) ListenForPeers() {
	udpConn, err := net.ListenUDP("udp4", g.udpAddr)
	if err != nil {
		fmt.Printf("Error: can't listen for peers on socket; %v", err)
		os.Exit(-1)
	}
	g.udpConn = udpConn

	fmt.Printf("Listening for peer communication on port %d\n", g.udpAddr.Port)
}

//SimpleHandleClientMessages has the gossiper listen for client messages and broadcast them (Simple mode)
func (g *gossiper) SimpleHandleClientMessages() {
	g.ConnectToClient()

	packetBytes := make([]byte, buffsize)
	msg := &message.Message{}

	for {
		nRead, _, err := g.clientConn.ReadFromUDP(packetBytes)
		if err != nil {
			fmt.Println("Error: read from buffer failed.")
			os.Exit(-1)
		}

		if nRead > 0 {
			protobuf.Decode(packetBytes, msg)
			printClientMessage(*msg)
			g.PrintPeers()
			simpleMsg := &message.SimpleMessage{OriginalName: g.name, RelayPeerAddr: g.udpAddr.String(), Contents: msg.Text}
			(*g).PrintPeers()
			packet := &gossippacket.GossipPacket{Simple: simpleMsg}
			g.broadcastSimpleMessage(packet, nil)
		}
	}
}

//printClientMessage prints client messages (should maybe be unexported?)
func printClientMessage(msg message.Message) {
	fmt.Println("CLIENT MESSAGE " + msg.Text)
}

//SimpleHandlePeersMessages has the gossiper listen for peer messages and broadcast them (Simple mode)
func (g *gossiper) SimpleHandlePeersMessages() {
	g.ListenForPeers()

	packetBytes := make([]byte, buffsize)
	packet := &gossippacket.GossipPacket{}

	for {
		nRead, _, err := g.udpConn.ReadFromUDP(packetBytes)
		if err != nil {
			fmt.Println("Error: read from buffer failed.")
			os.Exit(-1)
		}

		//debug
		if nRead > 0 {
			protobuf.Decode(packetBytes, packet)
			addr, err := net.ResolveUDPAddr("udp4", packet.Simple.RelayPeerAddr)
			if err != nil {
				fmt.Println("Error: couldn't resolve sender's address")
				os.Exit(-1)
			}
			if !g.isPeerKnown(addr) {
				g.AddPeer(addr)
			}
			printSimpleMessage(*packet.Simple)
			(*g).PrintPeers()
			packet.Simple.RelayPeerAddr = fmt.Sprintf("%v:%v", g.udpAddr.IP, g.udpAddr.Port)
			g.broadcastSimpleMessage(packet, addr)
		}
	}
}

//broadcastSimpleMessage sends a message to all connected peers, minus the sender (if the message doesn't come from the client)
func (g *gossiper) broadcastSimpleMessage(packet *gossippacket.GossipPacket, origin *net.UDPAddr) {
	for addr, conn := range g.peers {
		if origin != nil {
			if addr == origin.String() {
				continue
			}
		}
		fmt.Println("Sending to " + addr)
		packetBytes, err := protobuf.Encode(packet)
		if err != nil {
			fmt.Println("Error in encoding the message")
			os.Exit(-1)
		}

		_, err = conn.Write(packetBytes[0:])
		if err != nil {
			fmt.Printf("Error in sending the message. Error code: %v\n", err)
			os.Exit(-1)
		}
	}
}

//printSimpleMessage prints peer messages
func printSimpleMessage(msg message.SimpleMessage) {
	fmt.Printf("SIMPLE MESSAGE origin %s from %s contents %s\n", msg.OriginalName, msg.RelayPeerAddr, msg.Contents)
}

//printIncomingRumorMessage prints rumor messages after receiving them from another peer
func printIncomingRumorMessage(msg message.RumorMessage, relay *net.UDPAddr) {
	fmt.Printf("RUMOR origin %s from %s contents %s\n", msg.Origin, relay.String(), msg.Text)
}

//printOutgoingRumorMessage prints rumor messages before mongering
func printOutgoingRumorMessage(destination string) {
	fmt.Println("MONGERING with " + destination)
}

//printStatusPacket prints a status packet received
func printStatusPacket(status message.StatusPacket, relay *net.UDPAddr) {
	fmt.Printf("STATUS from %s", relay.String())
	for _, nameIDPair := range status.Want {
		fmt.Printf(" peer %s nextID %v", nameIDPair.Identifier, nameIDPair.NextID)
	}
	fmt.Printf("\n")
}

//printCoinFlipSuccess prints result of coin flip if rumor mongering keeps going after it
func printCoinFlipSuccess(destination *net.UDPAddr) {
	fmt.Println("FLIPPED COIN sending rumor to " + destination.String())
}

//printSyncWith prints a message when it's up to date with a peer it received a status message from
func printSyncWith(sender string) {
	fmt.Println("IN SYNC WITH ", sender)
}

/*
//deliverMessage sends a message to the client
func (g *gossiper) deliverMessage(packet *gossippacket.GossipPacket) {
	packetBytes, err := protobuf.Encode(&packet.Simple)
	if err != nil {
		fmt.Println("Error in encoding the message")
		os.Exit(-1)
	}

	n, err := g.clientConn.WriteToUDP(packetBytes, g.clientAddr)
	if err != nil {
		fmt.Printf("Error in sending the message. Error code: %v\n", err)
		os.Exit(-1)
	}
	fmt.Printf("%d bytes correctly sent!\n", n)
}
*/

func getInfosFromCL() (string, *net.UDPAddr, *net.UDPAddr, []*net.UDPAddr, bool) {
	gossiperName := flag.String("name", "my_gossiper", "Gossiper's name")
	clientPort := flag.String("UIPort", "8080", "Port to listen for the client")
	gossiperAddr := flag.String("gossipAddr", "127.0.0.1:5000", "Gossiper's IP and port")
	peersList := flag.String("peers", "", "List of peers' IP address and port")
	simpleMode := flag.Bool("simple", false, "Simple mode execution")

	flag.Parse()

	peersSlice := strings.Split(*peersList, ",")

	udpAddr, err := net.ResolveUDPAddr("udp4", *gossiperAddr)
	if err != nil {
		fmt.Println("Couldn't resolve gossiping UDP address")
		os.Exit(-1)
	}
	clientAddr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:"+*clientPort)
	if err != nil {
		fmt.Println("Couldn't resolve UDP address for client connection")
		os.Exit(-1)
	}

	var peersAddr [](*net.UDPAddr)
	for _, peer := range peersSlice {
		peerAddr, err := net.ResolveUDPAddr("udp4", peer)
		if err != nil {
			fmt.Println("Error in retrieving peer's address")
			os.Exit(-1)
		}
		peersAddr = append(peersAddr, peerAddr)
	}

	return *gossiperName, udpAddr, clientAddr, peersAddr, *simpleMode
}

func (g *gossiper) isPeerKnown(testPeerAddr *net.UDPAddr) bool {
	for addr := range g.peers {
		if testPeerAddr.String() == addr {
			return true
		}
	}

	return false
}

func (g gossiper) PrintPeers() {
	var peersSlice []string
	for peer := range g.peers {
		peersSlice = append(peersSlice, peer)
	}
	fmt.Println(strings.Join(peersSlice, ","))
}

func (g gossiper) isMessageKnown(rumorMsg message.RumorMessage) bool {
	return g.myStatus[rumorMsg.Origin] > rumorMsg.ID
}

func (g gossiper) selectRandomPeer() string { //Doesn't check that the selected peer is not the sender!!!
	i := rand.Intn(len(g.peers))
	var addr string
	for addr = range g.peers {
		i--
		if i == 0 {
			break
		}
	}

	return addr
}

func getPacketType(packet gossippacket.GossipPacket) int {
	if packet.Rumor != nil {
		return message.Rumor
	}
	if packet.Simple != nil {
		return message.Simple
	}
	if packet.Status != nil {
		return message.Status
	}

	return -1
}

func keepMongering() bool {
	return (rand.Int() % 2) == 1
}

func (g *gossiper) compareStatus(msg message.StatusPacket, sender string) status {
	needMessages := false

	for _, request := range msg.Want {
		if g.myStatus[request.Identifier] > request.NextID {
			gp := &gossippacket.GossipPacket{Simple: nil, Rumor: g.rumorMsgs[request.Identifier][request.NextID], Status: nil, RelayPeer: g.udpAddr.String()}
			g.sendPacket(gp, sender)
			fmt.Println("COMPARE RESULT: Sent subsequent packet")
			return have
		}
		if g.myStatus[request.Identifier] < request.NextID {
			fmt.Println("COMPARE RESULT: Need something from target")
			needMessages = true
		}
	}
	if needMessages {
		g.sendPacket(g.makeStatusPacket(), sender)
		return want
	}

	printSyncWith(sender)
	return equal
}

//HandleClientMessages receives messages from clients, encapsulate them as Rumor Messages (and gossip packets) and proceeds to rumor monger
func (g *gossiper) HandleClientMessages() {
	g.ConnectToClient()

	packetBytes := make([]byte, buffsize)
	msg := &message.Message{}

	for {
		nRead, _, err := g.clientConn.ReadFromUDP(packetBytes)
		if err != nil {
			fmt.Println("Error: read from buffer failed.")
			os.Exit(-1)
		}

		if nRead > 0 {
			protobuf.Decode(packetBytes, msg)
			printClientMessage(*msg)
			g.PrintPeers()

			rumorMsg := &message.RumorMessage{Origin: g.name, ID: messageID, Text: msg.Text}
			g.rumorMsgs[g.name] = append(g.rumorMsgs[g.name], rumorMsg)
			messageID++

			packet := &gossippacket.GossipPacket{Rumor: rumorMsg, RelayPeer: g.udpAddr.String()}
			go g.rumorMonger(packet, nil)
		}
	}
}

//HandlePeersMessages reads messages coming from peers; if they are acks, it tries to send them to the appropriate channel. If they are not, it starts a rumor mongering routine
func (g *gossiper) HandlePeersMessages() {
	g.ListenForPeers()

	packetBytes := make([]byte, buffsize)
	packet := &gossippacket.GossipPacket{}

	for {
		nRead, _, err := g.udpConn.ReadFromUDP(packetBytes)
		if err != nil {
			fmt.Println("Error: read from buffer failed.")
			os.Exit(-1)
		}

		if nRead > 0 {
			protobuf.Decode(packetBytes, packet)
			addr, err := net.ResolveUDPAddr("udp4", packet.RelayPeer)
			if err != nil {
				fmt.Println("Error: couldn't resolve sender's address")
				os.Exit(-1)
			}
			fmt.Println("Checking if peer is known")
			if !g.isPeerKnown(addr) {
				g.AddPeer(addr)
			}
			g.PrintPeers()
			packetType := getPacketType(*packet)
			switch packetType {
			case message.Simple:
				fmt.Println("Error. No simple message allowed outside of simple mode")
			case message.Rumor:
				printIncomingRumorMessage(*packet.Rumor, addr)

				g.sendPacket(g.makeStatusPacket(), addr.String())
				if !g.isMessageKnown(*packet.Rumor) {

					//Add unknown message to my list of messages and sort them
					g.rumorMsgs[packet.Rumor.Origin] = append(g.rumorMsgs[packet.Rumor.Origin], packet.Rumor)
					sort.Slice(g.rumorMsgs[packet.Rumor.Origin], func(i, j int) bool {
						return g.rumorMsgs[packet.Rumor.Origin][i].ID < g.rumorMsgs[packet.Rumor.Origin][j].ID
					})

					//If I have received the message I was waiting for, I update my status
					if packet.Rumor.ID == g.myStatus[packet.Rumor.Origin] {
						var nextID uint32
						for nextID = g.myStatus[packet.Rumor.Origin] + 1; nextID < uint32(len(g.rumorMsgs[packet.Rumor.Origin])); nextID++ {
							if g.rumorMsgs[packet.Rumor.Origin][nextID].ID != nextID {
								break
							}
						}

						g.myStatus[packet.Rumor.Origin] = nextID
					}

					go g.rumorMonger(packet, addr)
				} else {
					fmt.Println("Known message received. Ignoring.............")
				}
			case message.Status:
				printStatusPacket(*packet.Status, addr)
				if _, ok := g.channels[addr.String()]; ok { //Need control on possible empty array?
					/*for _, ch := range g.channels[addr.String()] {
						if packet.Rumor != nil {
							fmt.Println("I AM PUTTING SOMETHING WEIRD: " + packet.Rumor.Text)
						}
						ch <- packet
					}*/
				} else {
					g.compareStatus(*packet.Status, addr.String())
				}
			}
		}
	}
}

func (g *gossiper) rumorMonger(packet *gossippacket.GossipPacket, addr *net.UDPAddr) {
	//If message comes from peer, doesn't send it back to peer
	var contacted []string
	if addr != nil {
		contacted = append(contacted, addr.String())
	}

	for {
		//Select the random peer in a smart way
		if len(contacted) == len(g.peers) {
			fmt.Println("No peer to contact")
			return
		}
		peer := g.selectRandomPeer()
		valid := false
		for !valid {
			valid = true
			for _, contactedPeer := range contacted {
				if contactedPeer == peer {
					valid = false
					peer = g.selectRandomPeer()
					break
				}
			}
		}

		//Add a channel to get status message linked to  the peer
		c := make(chan *gossippacket.GossipPacket)
		g.channels[peer] = append(g.channels[peer], c)

		//At the end of the function, deletes the channel
		defer func() {
			for i, channel := range g.channels[peer] {
				if channel == c {
					g.channels[peer] = append(g.channels[peer][:i], g.channels[peer][i+1:]...)
				}
			}
		}()

		//Sends the packet to the selected peer
		printOutgoingRumorMessage(peer)
		packet.RelayPeer = g.udpAddr.String()
		g.sendPacket(packet, peer)

		//Checks if rumor mongering process is done or if it has to select another peer
		fmt.Printf("Checking if mongering is done for message: %v\n", packet.Rumor.Text)
		if g.isMongeringDone(c, peer) {
			return
		}

		contacted = append(contacted, peer)
	}

}

func (g gossiper) makeStatusPacket() *gossippacket.GossipPacket {
	var want []message.PeerStatus
	for id, val := range g.myStatus {
		mystat := message.PeerStatus{Identifier: id, NextID: val}
		want = append(want, mystat)
	}
	peerStatus := &message.StatusPacket{Want: want}
	gp := &gossippacket.GossipPacket{Simple: nil, Rumor: nil, Status: peerStatus, RelayPeer: g.udpAddr.String()}
	return gp
}

func (g gossiper) sendPacket(packet *gossippacket.GossipPacket, addr string) {
	packetBytes, err := protobuf.Encode(packet)
	if err != nil {
		fmt.Println("Error in encoding the message")
		os.Exit(-1)
	}

	_, err = g.peers[addr].Write(packetBytes[0:])
	if err != nil {
		fmt.Printf("Error in sending the message. Error code: %v\n", err)
		os.Exit(-1)
	}
}

func (g gossiper) isMongeringDone(c chan *gossippacket.GossipPacket, peer string) bool {
	ticker := time.NewTicker(timerLength * time.Second)

	select {
	case gp := <-c:
		if gp.Status == nil {
			fmt.Println("ERROR: NOT STATUS PACKET SENT IN STATUS CHANNEL?!")
			fmt.Println(gp.Rumor.Text)
			os.Exit(-1)
		}
		s := g.compareStatus(*gp.Status, peer)
		switch s {
		case have:
			return g.isMongeringDone(c, peer)
		case want:
			return true
		case equal:
			return !keepMongering()
		}
	case <-ticker.C:
		return false
	}

	return false
}

func (g *gossiper) AntiEntropy() {
	ticker := time.NewTicker(timerLength * time.Second)

	for {
		select {
		case <-ticker.C:
			g.sendPacket(g.makeStatusPacket(), g.selectRandomPeer())
		}
	}
}

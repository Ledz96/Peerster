package gossiper

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	"github.com/AlessandroBianchi/Peerster/utils"

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
	messageID = 1
	timerLength = 10
}

//Gossiper is a struct that contains all the information needed by the gossiper
type Gossiper struct {
	name              string
	udpAddr           *net.UDPAddr
	udpConn           *net.UDPConn
	clientAddr        *net.UDPAddr
	clientConn        *net.UDPConn
	peers             *utils.SafePeersMap
	simpleMode        bool
	myStatus          *utils.SafeStatusMap
	rumorMsgs         *utils.SafeMsgMap
	channels          *utils.SafeChanMap
	antiEntropyLength time.Duration
	newMsgs           []message.RumorMessage
	newNodes          []string
}

//New creates a new Gossiper
func New() *Gossiper {
	g := Gossiper{}
	g.peers = utils.NewSafePeersMap()
	g.myStatus = utils.NewSafeStatusMap()
	g.rumorMsgs = utils.NewSafeMsgMap()
	g.channels = utils.NewSafeChanMap()
	return &g
}

//Simple returns the simple bool variable, stating whether or not the program runs in simple mode
func (g Gossiper) Simple() bool {
	return g.simpleMode
}

//IsAntiEntropyActive checks if AntiEntroy is activated from command line
func (g Gossiper) IsAntiEntropyActive() bool {
	return g.antiEntropyLength > 0
}

//Name returns the name (ID) of the peer
func (g Gossiper) Name() string {
	return g.name
}

//ClientPort returns the client's port
func (g Gossiper) ClientPort() string {
	return fmt.Sprintf("%v", g.clientAddr.Port)
}

//SetInfos read infos from command line and stores them in the structure
func (g *Gossiper) SetInfos() {
	var peersAddr []*net.UDPAddr
	var antiEntropy int64

	g.name, g.udpAddr, g.clientAddr, peersAddr, g.simpleMode, antiEntropy = getInfosFromCL()

	if antiEntropy > 0 {
		g.antiEntropyLength = time.Duration(antiEntropy)
	}

	for _, peerAddr := range peersAddr {
		g.AddPeer(peerAddr)
	}
}

//AddPeer adds a peer to the Gossiper
func (g *Gossiper) AddPeer(peerAddr *(net.UDPAddr)) {
	peerConn, err := net.DialUDP("udp4", nil, peerAddr)
	if err != nil {
		fmt.Printf("Error: could not connect to given peer: %v", err)
		os.Exit(-1)
	}
	g.peers.AddPeer(peerAddr.String(), peerConn)

	g.newNodes = append(g.newNodes, peerAddr.String())
}

//ConnectToClient connects to the client
func (g *Gossiper) ConnectToClient() {
	fmt.Println("Attempting client connection")

	clientConn, err := net.ListenUDP("udp4", g.clientAddr)
	if err != nil {
		fmt.Printf("Error: can't listen for clients on socket; %v\n", err)
		os.Exit(-1)
	}
	g.clientConn = clientConn

	fmt.Printf("Listening for client on port %d\n", g.clientAddr.Port)
}

//ListenForPeers opens connections to peers
func (g *Gossiper) ListenForPeers() {
	udpConn, err := net.ListenUDP("udp4", g.udpAddr)
	if err != nil {
		fmt.Printf("Error: can't listen for peers on socket; %v", err)
		os.Exit(-1)
	}
	g.udpConn = udpConn

	fmt.Printf("Listening for peer communication on port %d\n", g.udpAddr.Port)
}

//SimpleHandleClientMessages has the Gossiper listen for client messages and broadcast them (Simple mode)
func (g *Gossiper) SimpleHandleClientMessages() {
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

//SimpleHandlePeersMessages has the Gossiper listen for peer messages and broadcast them (Simple mode)
func (g *Gossiper) SimpleHandlePeersMessages() {
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
func (g *Gossiper) broadcastSimpleMessage(packet *gossippacket.GossipPacket, origin *net.UDPAddr) {
	peersSlice := g.peers.GetSlice()
	for _, addr := range peersSlice {
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

		udpaddr, err := net.ResolveUDPAddr("udp4", addr)
		if err != nil {
			fmt.Println("Error in converting udp addr")
		}
		_, err = g.udpConn.WriteTo(packetBytes[0:], udpaddr)
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
	fmt.Printf("RUMOR origin %s from %s ID %v contents %s\n", msg.Origin, relay.String(), msg.ID, msg.Text)
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
func printCoinFlipSuccess(destination string) {
	fmt.Println("FLIPPED COIN sending rumor to " + destination)
}

//printSyncWith prints a message when it's up to date with a peer it received a status message from
func printSyncWith(sender string) {
	fmt.Println("IN SYNC WITH " + sender)
}

func getInfosFromCL() (string, *net.UDPAddr, *net.UDPAddr, []*net.UDPAddr, bool, int64) {
	gossiperName := flag.String("name", "my_gossiper", "Gossiper's name")
	clientPort := flag.String("UIPort", "8080", "Port to listen for the client")
	gossiperAddr := flag.String("gossipAddr", "127.0.0.1:5000", "Gossiper's IP and port")
	peersList := flag.String("peers", "", "List of peers' IP address and port")
	simpleMode := flag.Bool("simple", false, "Simple mode execution")
	antiEntropy := flag.Int64("antiEntropy", 10, "Length of anti-entropy interval")

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

	return *gossiperName, udpAddr, clientAddr, peersAddr, *simpleMode, *antiEntropy
}

func (g *Gossiper) isPeerKnown(testPeerAddr *net.UDPAddr) bool {
	return g.peers.IsPeerPresent(testPeerAddr.String())
}

//PrintPeers print the gossiper's peers
func (g Gossiper) PrintPeers() {
	peersSlice := g.peers.GetSlice()
	fmt.Println(strings.Join(peersSlice, ","))
}

func (g Gossiper) isMessageKnown(rumorMsg message.RumorMessage) bool {
	for _, msg := range g.rumorMsgs.GetMessagesByOrigin(rumorMsg.Origin) {
		if msg == rumorMsg {
			return true
		}
	}

	return false
}

func (g Gossiper) selectRandomPeer() string { //Doesn't check that the selected peer is not the sender!!!
	return g.peers.GetRandomPeer()
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

func (g *Gossiper) compareStatus(msg message.StatusPacket, sender string) status {
	needMessages := false

	for _, request := range msg.Want {
		if g.myStatus.GetStatus(request.Identifier) > request.NextID {
			rumor := g.rumorMsgs.GetMessageByIndex(request.Identifier, int(request.NextID-1))
			gp := &gossippacket.GossipPacket{Simple: nil, Rumor: &rumor, Status: nil}
			g.sendPacket(gp, sender)
			fmt.Println("COMPARE RESULT: Sent subsequent packet")
			return have
		}
		if g.myStatus.GetStatus(request.Identifier) < request.NextID {
			fmt.Println("COMPARE RESULT: Need something from target")
			needMessages = true
		}
	}
	originsList := g.myStatus.GetOriginsList()
	for _, origin := range originsList {
		present := false
		for _, val := range msg.Want {
			if val.Identifier == origin {
				present = true
			}
		}

		if !present {
			rumor := g.rumorMsgs.GetMessageByIndex(origin, 0)
			gp := &gossippacket.GossipPacket{Simple: nil, Rumor: &rumor, Status: nil}
			g.sendPacket(gp, sender)
			fmt.Println("COMPARE RESULT: Sent unknown packet")
			return have
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
func (g *Gossiper) HandleClientMessages() {
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

			rumorMsg := message.RumorMessage{Origin: g.name, ID: messageID, Text: msg.Text}
			g.rumorMsgs.AddMessage(g.name, rumorMsg)
			g.newMsgs = append(g.newMsgs, rumorMsg)
			messageID++
			g.myStatus.SetStatus(g.name, messageID)

			packet := &gossippacket.GossipPacket{Rumor: &rumorMsg}
			go g.rumorMonger(*packet, nil)
		}
	}
}

//HandlePeersMessages reads messages coming from peers; if they are acks, it tries to send them to the appropriate channel. If they are not, it starts a rumor mongering routine
func (g *Gossiper) HandlePeersMessages() {
	g.ListenForPeers()

	packetBytes := make([]byte, buffsize)
	packet := &gossippacket.GossipPacket{}

	for {
		nRead, addrComing, err := g.udpConn.ReadFromUDP(packetBytes)
		if err != nil {
			fmt.Println("Error: read from buffer failed.")
			os.Exit(-1)
		}

		if nRead > 0 {
			protobuf.Decode(packetBytes, packet)
			addr, err := net.ResolveUDPAddr("udp4", addrComing.String())
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

				if !g.isMessageKnown(*packet.Rumor) {

					//Add unknown message to my list of messages and sort them
					g.rumorMsgs.AddMessage(packet.Rumor.Origin, *packet.Rumor)
					g.rumorMsgs.SortMessages(packet.Rumor.Origin)
					g.newMsgs = append(g.newMsgs, *packet.Rumor)

					//If this is the first message I get, I update the one I'm looking for before doing the check
					if g.myStatus.GetStatus(packet.Rumor.Origin) == 0 {
						g.myStatus.SetStatus(packet.Rumor.Origin, 1)
					}
					//If I have received the message I was waiting for, I update my status
					if packet.Rumor.ID == g.myStatus.GetStatus(packet.Rumor.Origin) {
						var nextID uint32
						for nextID = g.myStatus.GetStatus(packet.Rumor.Origin); nextID < uint32(g.rumorMsgs.GetMessagesNumberByOrigin(packet.Rumor.Origin)); nextID++ {
							rumorMsg := g.rumorMsgs.GetMessageByIndex(packet.Rumor.Origin, int(nextID))
							if rumorMsg.ID != nextID+1 {
								break
							}
						}

						g.myStatus.SetStatus(packet.Rumor.Origin, nextID+1)
					}

					go g.rumorMonger(*packet, addr)
				}

				g.sendPacket(g.makeStatusPacket(), addr.String())
			case message.Status:
				printStatusPacket(*packet.Status, addr)
				if g.channels.GetChannelsNumberByPeer(addr.String()) > 0 {
					for _, ch := range g.channels.GetChannelsByPeer(addr.String()) {
						fmt.Printf("ACCESSING CHANNEL %v ; WRITING %v\n", ch, packet)
						ch <- *packet
					}
				} else {
					g.compareStatus(*packet.Status, addr.String())
				}
			}
		}
	}
}

func (g *Gossiper) rumorMonger(packet gossippacket.GossipPacket, addr *net.UDPAddr) {
	//If message comes from peer, doesn't send it back to peer
	var contacted []string
	if addr != nil {
		contacted = append(contacted, addr.String())
	}

	coin := false

	for {
		//Select the random peer in a smart way
		if len(contacted) == g.peers.GetNumber() {
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

		if coin {
			printCoinFlipSuccess(peer)
		}

		//Add a channel to get status message linked to  the peer
		c := make(chan gossippacket.GossipPacket, 1)
		g.channels.AddChannel(peer, c)

		//Sends the packet to the selected peer
		printOutgoingRumorMessage(peer)
		g.sendPacket(&packet, peer)

		//Checks if rumor mongering process is done or if it has to select another peer
		fmt.Printf("Checking if mongering is done for message: %v\n", packet.Rumor.Text)
		var done bool
		done, coin = g.isMongeringDone(c, peer)
		//At the end of the function, deletes the channel
		g.channels.DeleteChannel(peer, c)
		if done {
			return
		}

		contacted = append(contacted, peer)
	}

}

func (g Gossiper) makeStatusPacket() *gossippacket.GossipPacket {
	peerStatus := g.myStatus.GetStatusPacket()
	gp := &gossippacket.GossipPacket{Simple: nil, Rumor: nil, Status: &peerStatus}
	return gp
}

func (g Gossiper) sendPacket(packet *gossippacket.GossipPacket, addr string) {
	packetBytes, err := protobuf.Encode(packet)
	if err != nil {
		fmt.Println("Error in encoding the message")
		os.Exit(-1)
	}

	udpaddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		fmt.Println("Error in converting udp addr")
	}
	_, err = g.udpConn.WriteTo(packetBytes[0:], udpaddr)
	if err != nil {
		fmt.Printf("Error in sending the message. Error code: %v\n", err)
		os.Exit(-1)
	}
}

//isMongeringDone gets the ack, handles it and then tells the caller if the message must be mongered again. In case it does, the second boolean tells if it happened because of a coin flip
func (g Gossiper) isMongeringDone(c chan gossippacket.GossipPacket, peer string) (bool, bool) {
	ticker := time.NewTicker(timerLength * time.Second)

	select {
	case gp := <-c:
		fmt.Printf("ACCESSED CHANNEL %v ; READING %v\n", c, gp)
		s := g.compareStatus(*gp.Status, peer)
		switch s {
		case have:
			return g.isMongeringDone(c, peer)
		case want:
		case equal:
			if keepMongering() {
				return false, true
			}
			return true, false
		}
	case <-ticker.C:
		fmt.Println("TIMEOUT! No message received!")
		return false, false
	}

	return false, false
}

//AntiEntropy implements an AntiEntropy protocol
func (g *Gossiper) AntiEntropy() {
	ticker := time.NewTicker(g.antiEntropyLength * time.Second)

	for {
		select {
		case <-ticker.C:
			peer := g.selectRandomPeer()
			fmt.Println("Firing Anti-Entropy Message towards " + peer + "!")
			g.sendPacket(g.makeStatusPacket(), peer)
		}
	}
}

//Note: more efficient way to do this: when getting a new node/message, add it to a list. When asking for the new ones, empty the list...

//GetLatestRumorMessagesList returns a list of messages that were not returned in a previous call of the same function
func (g *Gossiper) GetLatestRumorMessagesList() []message.RumorMessage {
	defer func() {
		g.newMsgs = nil
	}()

	return g.newMsgs
}

//GetLatestNodesList returns a list of nodes that were not returned in a previous call of the same function
func (g *Gossiper) GetLatestNodesList() []string {
	defer func() {
		g.newNodes = nil
	}()

	return g.newNodes
}

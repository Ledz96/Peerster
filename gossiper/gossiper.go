package gossiper

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"

	gossippacket "github.com/AlessandroBianchi/Peerster/gossippacket"
	"github.com/AlessandroBianchi/Peerster/message"
	"github.com/dedis/protobuf"
)

const buffsize int = 1024

type gossiper struct {
	name       string
	udpAddr    *net.UDPAddr
	udpConn    *net.UDPConn
	clientAddr *net.UDPAddr
	clientConn *net.UDPConn
	peers      map[*net.UDPAddr](*net.UDPConn)
	simpleMode bool
}

//New creates a new gossiper
func New() *gossiper {
	g := gossiper{}
	g.peers = make(map[*net.UDPAddr](*net.UDPConn))
	return &g
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
		g.peers[peerAddr] = peerConn
	}
}

//AddPeer adds a peer to the gossiper
func (g *gossiper) AddPeer(peerAddr *(net.UDPAddr)) {
	peerConn, err := net.DialUDP("udp4", nil, peerAddr)
	if err != nil {
		fmt.Println("Error: could not connect to given peer")
		os.Exit(-1)
	}
	g.peers[peerAddr] = peerConn
}

//ListenForClients has the gossiper listen for client messages and store them (or just print them?)
func (g *gossiper) ListenForClients() {
	fmt.Println("Attempting client connection")

	clientConn, err := net.ListenUDP("udp4", g.clientAddr)
	if err != nil {
		fmt.Printf("Error: can't listen for clients on socket; %v\n", err)
		os.Exit(-1)
	}
	g.clientConn = clientConn

	fmt.Printf("Listening for client on port %d\n", g.clientAddr.Port)

	packetBytes := make([]byte, buffsize)
	msg := &message.SimpleMessage{}

	for {
		nRead, _, err := g.clientConn.ReadFromUDP(packetBytes)
		if err != nil {
			fmt.Println("Error: read from buffer failed.")
			os.Exit(-1)
		}

		if nRead > 0 {
			protobuf.Decode(packetBytes, msg)
			printClientMessage(*msg)
			(*g).PrintPeers()
			packet := &gossippacket.GossipPacket{Simple: msg}
			msg.OriginalName = g.name
			msg.RelayPeerAddr = fmt.Sprintf("%v:%v", g.udpAddr.IP, g.udpAddr.Port)
			g.broadcastMessage(packet, nil)
		} else {
			fmt.Println("Couldn't read anything...")
		}
	}
}

//printClientMessage prints client messages (should maybe be unexported?)
func printClientMessage(msg message.SimpleMessage) {
	fmt.Println("CLIENT MESSAGE " + msg.Contents)
}

//ListenForPeers has the gossiper listen for peer messages and store them (or just print them?)
func (g *gossiper) ListenForPeers() {
	udpConn, err := net.ListenUDP("udp4", g.udpAddr)
	if err != nil {
		fmt.Printf("Error: can't listen for peers on socket; %v", err)
		os.Exit(-1)
	}
	g.udpConn = udpConn

	fmt.Printf("Listening for peer communication on port %d\n", g.udpAddr.Port)

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
			printPeerMessage(*packet.Simple)
			(*g).PrintPeers()
			packet.Simple.RelayPeerAddr = fmt.Sprintf("%v:%v", g.udpAddr.IP, g.udpAddr.Port)
			g.broadcastMessage(packet, addr)
		}
	}
}

//broadcastMessage sends a message to all connected peers, minus the sender (if the message doesn't come from the client)
func (g *gossiper) broadcastMessage(packet *gossippacket.GossipPacket, origin *net.UDPAddr) {
	for addr, conn := range g.peers {
		if origin != nil {
			if isSameAddress(addr, origin) {
				continue
			}
		}
		fmt.Printf("Sending to %v:%v\n", addr.IP, addr.Port)
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

//printPeerMessage prints peer messages (should maybe be unexported?)
func printPeerMessage(msg message.SimpleMessage) {
	fmt.Printf("SIMPLE MESSAGE origin %s from %s contents %s\n", msg.OriginalName, msg.RelayPeerAddr, msg.Contents)
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
		if fmt.Sprintf("%v:%v", addr.IP, addr.Port) == fmt.Sprintf("%v:%v", testPeerAddr.IP, testPeerAddr.Port) {
			return true
		}
	}

	fmt.Println("Adding peer!")
	return false
}

func isSameAddress(addr1 *net.UDPAddr, addr2 *net.UDPAddr) bool {
	return fmt.Sprintf("%v:%v", addr1.IP, addr1.Port) == fmt.Sprintf("%v:%v", addr2.IP, addr2.Port)
}

func (g gossiper) PrintPeers() {
	for peer := range g.peers {
		fmt.Printf("%v:%v\n", peer.IP, peer.Port)
	}
}

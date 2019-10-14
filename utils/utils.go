package utils

import (
	"math/rand"
	"net"
	"sort"
	"sync"

	"github.com/AlessandroBianchi/Peerster/gossippacket"
	"github.com/AlessandroBianchi/Peerster/message"
)

//GetGossiperInfos parses from the command line to return gossiper's Name, UIPort, IP and listening port (move to Gossiper?)

//SafePeersMap implements a thread-safe map of peers
type SafePeersMap struct {
	v   map[string](*net.UDPConn)
	mux sync.Mutex
}

//NewSafePeersMap returns a new safe peers map
func NewSafePeersMap() *SafePeersMap {
	pm := SafePeersMap{}
	pm.v = make(map[string](*net.UDPConn))
	return &pm
}

//AddPeer adds a peer to the map
func (pm *SafePeersMap) AddPeer(addr string, conn *net.UDPConn) {
	pm.mux.Lock()
	pm.v[addr] = conn
	pm.mux.Unlock()
}

//GetPeer gets a peer from the map
func (pm *SafePeersMap) GetPeer(addr string) *net.UDPConn {
	pm.mux.Lock()
	defer pm.mux.Unlock()
	return pm.v[addr]
}

//IsPeerPresent checks if a peer is in the map
func (pm *SafePeersMap) IsPeerPresent(addr string) bool {
	pm.mux.Lock()
	_, ok := pm.v[addr]
	pm.mux.Unlock()
	return ok
}

//GetSlice gets the peers as a slice
func (pm *SafePeersMap) GetSlice() []string {
	var peersSlice []string

	pm.mux.Lock()
	for peer := range pm.v {
		peersSlice = append(peersSlice, peer)
	}
	pm.mux.Unlock()

	return peersSlice
}

//GetNumber returns the number of connected peers
func (pm *SafePeersMap) GetNumber() int {
	pm.mux.Lock()
	defer pm.mux.Unlock()
	return len(pm.v)
}

//GetRandomPeer returns a random peer
func (pm *SafePeersMap) GetRandomPeer() string {
	pm.mux.Lock()

	i := rand.Intn(len(pm.v))
	var addr string
	for addr = range pm.v {
		i--
		if i == 0 {
			break
		}
	}

	pm.mux.Unlock()

	return addr
}

//SafeStatusMap implements a thread-safe map of status
type SafeStatusMap struct {
	v   map[string]uint32
	mux sync.Mutex
}

//NewSafeStatusMap returns a new safe status map
func NewSafeStatusMap() *SafeStatusMap {
	sm := SafeStatusMap{}
	sm.v = make(map[string]uint32)
	return &sm
}

//GetStatus returns the status value of an origin
func (sm *SafeStatusMap) GetStatus(origin string) uint32 {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	return sm.v[origin]
}

//SetStatus sets a status value for an origin
func (sm *SafeStatusMap) SetStatus(origin string, value uint32) {
	sm.mux.Lock()
	sm.v[origin] = value
	sm.mux.Unlock()
}

//GetOriginsList gets list of all the origins in the status map (peers of whom I got messages)
func (sm *SafeStatusMap) GetOriginsList() []string {
	var origins []string

	sm.mux.Lock()
	for origin := range sm.v {
		origins = append(origins, origin)
	}
	sm.mux.Unlock()

	return origins
}

//GetStatusPacket returns a status packet from a status map
func (sm *SafeStatusMap) GetStatusPacket() message.StatusPacket {
	sm.mux.Lock()

	var want []message.PeerStatus
	for id, val := range sm.v {
		mystat := message.PeerStatus{Identifier: id, NextID: val}
		want = append(want, mystat)
	}

	sm.mux.Unlock()

	statusPacket := message.StatusPacket{Want: want}

	return statusPacket
}

//SafeMsgMap implements a thread-safe map of rumor msgs
type SafeMsgMap struct {
	v   map[string][]message.RumorMessage
	mux sync.Mutex
}

//NewSafeMsgMap returns a safe map of messages
func NewSafeMsgMap() *SafeMsgMap {
	mm := SafeMsgMap{}
	mm.v = make(map[string][]message.RumorMessage)
	return &mm
}

//GetMessageByIndex returns a single message, given its index and origin
func (mm *SafeMsgMap) GetMessageByIndex(origin string, index int) message.RumorMessage {
	mm.mux.Lock()
	defer mm.mux.Unlock()
	return mm.v[origin][index]
}

//GetMessagesByOrigin returns all messages from a single origin
func (mm *SafeMsgMap) GetMessagesByOrigin(origin string) []message.RumorMessage {
	mm.mux.Lock()
	defer mm.mux.Unlock()
	return mm.v[origin]
}

//AddMessage adds a message to the map
func (mm *SafeMsgMap) AddMessage(origin string, msg message.RumorMessage) {
	mm.mux.Lock()
	mm.v[origin] = append(mm.v[origin], msg)
	mm.mux.Unlock()
}

//SortMessages sorts all messages from a certain origin
func (mm *SafeMsgMap) SortMessages(origin string) {
	mm.mux.Lock()
	sort.Slice(mm.v[origin], func(i, j int) bool {
		return mm.v[origin][i].ID < mm.v[origin][j].ID
	})
	mm.mux.Unlock()
}

//GetMessagesNumberByOrigin gets the number of message from a certain origin
func (mm *SafeMsgMap) GetMessagesNumberByOrigin(origin string) int {
	mm.mux.Lock()
	defer mm.mux.Unlock()
	return len(mm.v[origin])
}

//SafeChanMap implements a thread-safe map of Channels
type SafeChanMap struct {
	v   map[string][]chan gossippacket.GossipPacket
	mux sync.Mutex
}

//NewSafeChanMap creates a new safe map of channels
func NewSafeChanMap() *SafeChanMap {
	cm := SafeChanMap{}
	cm.v = make(map[string][]chan gossippacket.GossipPacket)
	return &cm
}

//GetChannelsByPeer returns all the channels with a certain peer
func (cm *SafeChanMap) GetChannelsByPeer(peer string) []chan gossippacket.GossipPacket {
	cm.mux.Lock()
	defer cm.mux.Unlock()
	return cm.v[peer]
}

//GetChannelsNumberByPeer returns the number of channels connecting with a certain peer
func (cm *SafeChanMap) GetChannelsNumberByPeer(peer string) int {
	cm.mux.Lock()
	defer cm.mux.Unlock()
	return len(cm.v[peer])
}

//AddChannel adds a channel for a certain peer
func (cm *SafeChanMap) AddChannel(peer string, c chan gossippacket.GossipPacket) {
	cm.mux.Lock()
	cm.v[peer] = append(cm.v[peer], c)
	cm.mux.Unlock()
}

//DeleteChannel deletes a channel from the map
func (cm *SafeChanMap) DeleteChannel(peer string, c chan gossippacket.GossipPacket) {
	cm.mux.Lock()
	for i, channel := range cm.v[peer] {
		if channel == c {
			cm.v[peer] = append(cm.v[peer][:i], cm.v[peer][i+1:]...)
			break
		}
	}
	cm.mux.Unlock()
}

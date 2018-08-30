package peer

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	mrand "math/rand"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-host"
	"github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/ninjadotorg/cash-prototype/common"
	"github.com/ninjadotorg/cash-prototype/wire"
)

const (
	LOCAL_HOST = "127.0.0.1"
	// trickleTimeout is the duration of the ticker which trickles down the
	// inventory to a peer.
	trickleTimeout = 10 * time.Second
)

// ConnState represents the state of the requested connection.
type ConnState uint8

// ConnState can be either pending, established, disconnected or failed.  When
// a new connection is requested, it is attempted and categorized as
// established or failed depending on the connection result.  An established
// connection which was disconnected is categorized as disconnected.
const (
	ConnPending ConnState = iota
	ConnFailing
	ConnCanceled
	ConnEstablished
	ConnDisconnected
)

type Peer struct {
	Host           host.Host

	TargetAddress    ma.Multiaddr
	PeerId           peer.ID
	RawAddress       string
	ListeningAddress common.SimpleAddr

	Seed      int64
	FlagMutex sync.Mutex
	Config    Config

	PeerConns map[peer.ID]*PeerConn
}

// Config is the struct to hold configuration options useful to Peer.
type Config struct {
	MessageListeners MessageListeners
}

type WrappedStream struct {
	Stream net.Stream
	Writer *bufio.Writer
	Reader *bufio.Reader
}

// MessageListeners defines callback function pointers to invoke with message
// listeners for a peer. Any listener which is not set to a concrete callback
// during peer initialization is ignored. Execution of multiple message
// listeners occurs serially, so one callback blocks the execution of the next.
//
// NOTE: Unless otherwise documented, these listeners must NOT directly call any
// blocking calls (such as WaitForShutdown) on the peer instance since the input
// handler goroutine blocks until the callback has completed.  Doing so will
// result in a deadlock.
type MessageListeners struct {
	OnTx        func(p *PeerConn, msg *wire.MessageTx)
	OnBlock     func(p *PeerConn, msg *wire.MessageBlock)
	OnGetBlocks func(p *PeerConn, msg *wire.MessageGetBlocks)
	OnVersion   func(p *PeerConn, msg *wire.MessageVersion)
	OnVerAck    func(p *PeerConn, msg *wire.MessageVerAck)
	OnGetAddr   func(p *PeerConn, msg *wire.MessageGetAddr)
	OnAddr      func(p *PeerConn, msg *wire.MessageAddr)

	//PoS

	// OnVerAck    func(p *PeerConn, msg *wire.MessageVerAck)
}

// outMsg is used to house a message to be sent along with a channel to signal
// when the message has been sent (or won't be sent due to things such as
// shutdown)
type outMsg struct {
	msg      wire.Message
	doneChan chan<- struct{}
	//encoding wire.MessageEncoding
}

func (self Peer) NewPeer() (*Peer, error) {
	// If the seed is zero, use real cryptographic randomness. Otherwise, use a
	// deterministic randomness source to make generated keys stay the same
	// across multiple runs
	var r io.Reader
	if self.Seed == 0 {
		r = rand.Reader
	} else {
		r = mrand.New(mrand.NewSource(self.Seed))
	}

	// Generate a key pair for this Host. We will use it
	// to obtain a valid Host ID.
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return &self, err
	}

	ip := strings.Split(self.ListeningAddress.String(), ":")[0]
	if len(ip) == 0 {
		ip = LOCAL_HOST
	}
	port := strings.Split(self.ListeningAddress.String(), ":")[1]
	net := self.ListeningAddress.Network()
	listeningAddressString := fmt.Sprintf("/%s/%s/tcp/%s", net, ip, port)
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(listeningAddressString),
		libp2p.Identity(priv),
	}

	basicHost, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		return &self, err
	}

	// Build Host multiaddress
	mulAddrStr := fmt.Sprintf("/ipfs/%s", basicHost.ID().Pretty())

	hostAddr, err := ma.NewMultiaddr(mulAddrStr)
	if err != nil {
		log.Print(err)
		return &self, err
	}

	// Now we can build a full multiaddress to reach this Host
	// by encapsulating both addresses:
	addr := basicHost.Addrs()[0]
	fullAddr := addr.Encapsulate(hostAddr)
	Logger.log.Infof("I am listening on %s with PEER ID - %s\n", fullAddr, basicHost.ID().String())
	pid, err := fullAddr.ValueForProtocol(ma.P_IPFS)
	if err != nil {
		log.Print(err)
		return &self, err
	}
	peerid, err := peer.IDB58Decode(pid)
	if err != nil {
		log.Print(err)
		return &self, err
	}

	self.RawAddress = fullAddr.String()
	self.Host = basicHost
	self.TargetAddress = fullAddr
	self.PeerId = peerid
	//self.sendMessageQueue = make(chan outMsg, 1)
	return &self, nil
}

func (self Peer) Start() error {
	Logger.log.Info("Peer start")
	Logger.log.Info("Set stream handler and wait for connection from other peer")
	self.Host.SetStreamHandler("/blockchain/1.0.0", self.HandleStream)
	select {} // hang forever
	return nil
}

func (self Peer) NewPeerConnection(peerId peer.ID) (*PeerConn, error) {
	Logger.log.Infof("Opening stream to PEER ID - %s \n", self.PeerId.String())

	stream, err := self.Host.NewStream(context.Background(), peerId, "/blockchain/1.0.0")
	if err != nil {
		Logger.log.Errorf("Fail in opening stream to PEER ID - %s with err: %s", self.PeerId.String(), err.Error())
		return nil, err
	}

	remotePeerId := stream.Conn().RemotePeer()

	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	//self.InboundReaderWriterStreams[remotePeerId] = rw

	//go self.InMessageHandler(rw)
	//go self.OutMessageHandler(rw)

	peerConn := PeerConn{
		IsOutbound:         true,
		Peer:               &self,
		Config:             self.Config,
		PeerId:             remotePeerId,
		ReaderWriterStream: rw,
		quit:               make(chan struct{}),
		sendMessageQueue:   make(chan outMsg, 1),
		HandleConnected:    self.handleConnected,
		HandleDisconnected: self.handleDisconnected,
		HandleFailed:       self.handleFailed,
	}

	self.PeerConns[peerConn.PeerId] = &peerConn

	go peerConn.InMessageHandler(rw)
	go peerConn.OutMessageHandler(rw)

	peerConn.RetryCount = 0
	peerConn.updateState(ConnEstablished)

	return &peerConn, nil
}

func (self *Peer) HandleStream(stream net.Stream) {
	// Remember to close the stream when we are done.
	//defer stream.Close()

	remotePeerId := stream.Conn().RemotePeer()
	Logger.log.Infof("PEER %s Received a new stream from OTHER PEER with ID-%s", self.Host.ID().String(), remotePeerId.String())

	// TODO this code make EOF for libp2p
	//if !atomic.CompareAndSwapInt32(&self.connected, 0, 1) {
	//	return
	//}

	// Create a buffer stream for non blocking read and write.
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	//self.InboundReaderWriterStreams[remotePeerId] = rw

	//go self.InMessageHandler(rw)
	//go self.OutMessageHandler(rw)

	peerConn := PeerConn{
		IsOutbound:         false,
		Peer:               self,
		Config:             self.Config,
		PeerId:             remotePeerId,
		ReaderWriterStream: rw,
		quit:               make(chan struct{}),
		sendMessageQueue:   make(chan outMsg, 1),
		HandleConnected:    self.handleConnected,
		HandleDisconnected: self.handleDisconnected,
		HandleFailed:       self.handleFailed,
	}

	self.PeerConns[peerConn.PeerId] = &peerConn

	go peerConn.InMessageHandler(rw)
	go peerConn.OutMessageHandler(rw)

	peerConn.RetryCount = 0
	peerConn.updateState(ConnEstablished)
}

// QueueMessageWithEncoding adds the passed bitcoin message to the peer send
// queue. This function is identical to QueueMessage, however it allows the
// caller to specify the wire encoding type that should be used when
// encoding/decoding blocks and transactions.
//
// This function is safe for concurrent access.
func (self Peer) QueueMessageWithEncoding(msg wire.Message, doneChan chan<- struct{}) {
	for _, peerConnection := range self.PeerConns {
		peerConnection.QueueMessageWithEncoding(msg, doneChan)
	}
}

// negotiateOutboundProtocol sends our version message then waits to receive a
// version message from the peer.  If the events do not occur in that order then
// it returns an error.
func (self *Peer) NegotiateOutboundProtocol(peer *Peer) error {
	// Generate a unique nonce for this peer so self connections can be
	// detected.  This is accomplished by adding it to a size-limited map of
	// recently seen nonces.
	msg, err := wire.MakeEmptyMessage(wire.CmdVersion)
	msg.(*wire.MessageVersion).Timestamp = time.Unix(time.Now().Unix(), 0)
	msg.(*wire.MessageVersion).LocalAddress = self.ListeningAddress
	msg.(*wire.MessageVersion).RawLocalAddress = self.RawAddress
	msg.(*wire.MessageVersion).LocalPeerId = self.PeerId
	msg.(*wire.MessageVersion).RemoteAddress = peer.ListeningAddress
	msg.(*wire.MessageVersion).RawRemoteAddress = peer.RawAddress
	msg.(*wire.MessageVersion).RemotePeerId = peer.PeerId
	msg.(*wire.MessageVersion).LastBlock = 0
	msg.(*wire.MessageVersion).ProtocolVersion = 1
	if err != nil {
		return err
	}
	msgVersion, err := msg.JsonSerialize()
	if err != nil {
		return err
	}
	header := make([]byte, wire.MessageHeaderSize)
	CmdType, _ := wire.GetCmdType(reflect.TypeOf(msg))
	copy(header[:], []byte(CmdType))
	msgVersion = append(msgVersion, header...)
	msgVersionStr := hex.EncodeToString(msgVersion)
	msgVersionStr += "\n"
	Logger.log.Infof("Send a msgVersion: %s", msgVersionStr)
	rw := self.PeerConns[peer.PeerId].ReaderWriterStream
	self.FlagMutex.Lock()
	rw.Writer.WriteString(msgVersionStr)
	rw.Writer.Flush()
	self.FlagMutex.Unlock()
	return nil
}

func (p *Peer) Disconnect() {
}

func (p *Peer) handleConnected(peerConn *PeerConn) {
	Logger.log.Infof("handleConnected %s", peerConn.PeerId.String())

	peerConn.RetryCount = 0
}

func (p *Peer) handleDisconnected(peerConn *PeerConn) {
	if peerConn.IsOutbound {
		time.AfterFunc(10*time.Second, func() {
			Logger.log.Infof("Retry New Peer Connection %s", peerConn.PeerId.String())
			peerConn.RetryCount += 1
			peerConn.updateState(ConnPending)
			p.NewPeerConnection(peerConn.PeerId)
		})
	} else {
		_peerConn, ok := p.PeerConns[peerConn.PeerId]
		if ok {
			delete(p.PeerConns, _peerConn.PeerId)
		}
	}
}

func (p *Peer) handleFailed(peerConn *PeerConn) {
	_peerConn, ok := p.PeerConns[peerConn.PeerId]
	if ok {
		delete(p.PeerConns, _peerConn.PeerId)
	}
}

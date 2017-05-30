package listenserver

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"io/ioutil"
	"net"

	log "github.com/sirupsen/logrus"

	"github.com/piot/hasty-protocol/handler"
	"github.com/piot/hasty-protocol/packet"
	"github.com/piot/hasty-protocol/packetdeserializers"
)

const (
	CONN_TYPE = "tcp"
)

type Server struct {
	nextConnectionIdentity uint
}

func NewServer() Server {
	return Server{}
}

func setupCert(cfg *tls.Config, cert string, certPrivateKey string) error {

	cfg.RootCAs = x509.NewCertPool()
	ca, err := ioutil.ReadFile("certs/ca.pem")
	if err == nil {
		cfg.RootCAs.AppendCertsFromPEM(ca)
	}

	keyPair, err := tls.LoadX509KeyPair(cert, certPrivateKey)
	if err != nil {
		log.Warnf("server: loadkeys: %s", err)
		return err
	}
	cfg.Certificates = append(cfg.Certificates, keyPair)

	return nil
}

func (in *Server) Listen(listenerHandler Listener, host string, cert string, certPrivateKey string) error { // Listen for incoming connections.

	log.Infof("Listening to", host)
	config := new(tls.Config)
	certErr := setupCert(config, cert, certPrivateKey)
	if certErr != nil {
		log.Warnf("Couldn't load certs '%s'", certErr)
		return certErr
	}
	listener, err := tls.Listen(CONN_TYPE, host, config)
	if err != nil {
		log.Warnf("Error listening:", err.Error())
		return err
	}
	// Close the listener when the application closes.
	defer listener.Close()

	in.accepting(listener, listenerHandler)
	return nil
}

func (server *Server) accepting(listener net.Listener, listenerHandler Listener) {
	for {
		// Listen for an incoming connection.
		log.Infof("Waiting for accept...")
		conn, err := listener.Accept()
		if err != nil {
			log.Warnf("Error accepting: ", err)
		}
		server.nextConnectionIdentity++
		connectionIdentity := packet.NewConnectionID(server.nextConnectionIdentity)
		connection, _ := listenerHandler.CreateConnection(&conn, connectionIdentity)
		// Handle connections in a new goroutine.
		go handleConnection(connection, conn, connectionIdentity)
	}
}

// Handles incoming requests.
func handleConnection(delegator handler.PacketHandler, conn net.Conn, connectionIdentity packet.ConnectionID) {
	// Make a buffer to hold incoming data.
	// buf := make([]byte, 4096)
	defer conn.Close()
	log.Infof("Received a connection! '%s'", conn.RemoteAddr())
	temp := make([]byte, 1024)

	stream := packet.NewPacketStream(connectionIdentity)

	// l := log.New(os.Stderr, "", 0)

	for true {
		// Read the incoming connection into the buffer.
		n, err := conn.Read(temp)
		if err != nil {
			log.Warnf("%s Error reading: '%s'. Closing...", connectionIdentity, err)
			delegator.HandleTransportDisconnect()
			return
		}
		data := temp[:n]

		if false {
			hexPayload := hex.Dump(data)
			log.Debugf("%s TransportReceived: %s", connectionIdentity, hexPayload)
		}
		stream.Feed(data)
		newPacket, fetchErr := stream.FetchPacket()
		if fetchErr != nil {
			_, isNotDoneError := fetchErr.(*packet.PacketNotDoneError)
			if isNotDoneError {
			} else {
				log.Warnf("Fetcherror:%s", fetchErr)
			}
		} else {
			if newPacket.Payload() != nil {
				err := packetdeserializers.Deserialize(newPacket, delegator)
				if err != nil {
					log.Warnf("Deserialize error: '%s'", err)
					return
				}
			}
		}
	}
}

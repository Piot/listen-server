package listenserver

import (
	"net"

	"github.com/piot/hasty-protocol/handler"
	"github.com/piot/hasty-protocol/packet"
)

type Listener interface {
	CreateConnection(connection *net.Conn, connectionIdentity packet.ConnectionID) (handler.PacketHandler, error)
}

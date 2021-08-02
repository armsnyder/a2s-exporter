package testserver

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"net"

	"github.com/rumblefrog/go-a2s"
)

var challenge = uint32(1876276358)

// TestServer implements the A2S server spec.
// See: https://developer.valvesoftware.com/wiki/Server_queries
type TestServer struct {
	ServerInfo *a2s.ServerInfo
	PlayerInfo *a2s.PlayerInfo
}

// Serve runs the A2S server.
// The function blocks execution until an error is encountered.
func (t *TestServer) Serve(conn net.PacketConn) error {
	var buf [a2s.MaxPacketSize]byte

	for {
		// Read the next request packet.
		_, remoteAddr, err := conn.ReadFrom(buf[:])
		if err != nil {
			return err
		}

		// Handle the request packet.

		// Validate packet header.
		if binary.LittleEndian.Uint32(buf[:5]) != math.MaxUint32 {
			continue
		}

		queryType := buf[4]
		out := &udpWriter{conn: conn, addr: remoteAddr}

		switch queryType {
		// Server info query (no challenge).
		case 'T':
			err = t.writeServerInfo(out)

			// Player info query.
		case 'U':
			gotChallenge := binary.LittleEndian.Uint32(buf[5:9])

			switch gotChallenge {
			// No challenge.
			case math.MaxUint32:
				err = t.writeChallenge(out)

			// Correct challenge.
			case challenge:
				err = t.writePlayerInfo(out)
			}
		}

		if err != nil {
			return err
		}
	}
}

func (t *TestServer) writeChallenge(out io.Writer) error {
	var outBuf [9]byte
	binary.LittleEndian.PutUint32(outBuf[:4], math.MaxUint32)
	outBuf[4] = 'A'
	binary.LittleEndian.PutUint32(outBuf[5:], challenge)
	_, err := out.Write(outBuf[:])
	return err
}

func (t *TestServer) writeServerInfo(out io.Writer) error {
	info := t.ServerInfo
	if info == nil {
		info = &a2s.ServerInfo{}
	}

	// Response packet buffer.
	buf := &packetBuffer{}

	// Header.
	buf.WriteUInt32(math.MaxUint32)
	buf.WriteByte('I')

	// Payload.
	buf.WriteByte(info.Protocol)
	buf.WriteCString(info.Name)
	buf.WriteCString(info.Map)
	buf.WriteCString(info.Folder)
	buf.WriteCString(info.Game)
	buf.WriteUInt16(info.ID)
	buf.WriteByte(info.Players)
	buf.WriteByte(info.MaxPlayers)
	buf.WriteByte(info.Bots)
	buf.WriteByte(formatServerType(info.ServerType))
	buf.WriteByte(formatServerOS(info.ServerOS))
	buf.WriteBool(info.Visibility)
	buf.WriteBool(info.VAC)

	if a2s.AppID(info.ID) == a2s.App_TheShip {
		theShip := info.TheShip
		if info.TheShip == nil {
			theShip = &a2s.TheShipInfo{}
		}
		buf.WriteByte(formatTheShipMode(theShip.Mode))
		buf.WriteByte(theShip.Witnesses)
		buf.WriteByte(theShip.Duration)
	}

	buf.WriteCString(info.Version)

	// We will add to the EDF flag as we read the source data.
	var edf byte

	// Create a new buffer for EDF data so that it can be written after the EDF flag.
	edfBuf := &packetBuffer{}

	if info.ExtendedServerInfo != nil {
		if info.ExtendedServerInfo.Port != 0 {
			edf |= 0x80
			edfBuf.WriteUInt16(info.ExtendedServerInfo.Port)
		}
		if info.ExtendedServerInfo.SteamID != 0 {
			edf |= 0x10
			edfBuf.WriteUInt64(info.ExtendedServerInfo.SteamID)
		}
	}

	if info.SourceTV != nil {
		edf |= 0x40
		edfBuf.WriteUInt16(info.SourceTV.Port)
		edfBuf.WriteCString(info.SourceTV.Name)
	}

	if info.ExtendedServerInfo != nil {
		if info.ExtendedServerInfo.Keywords != "" {
			edf |= 0x20
			edfBuf.WriteCString(info.ExtendedServerInfo.Keywords)
		}
		if info.ExtendedServerInfo.GameID != 0 {
			edf |= 0x01
			edfBuf.WriteUInt64(info.ExtendedServerInfo.GameID)
		}
	}

	if edf != 0 {
		// Write the EDF flag first.
		buf.WriteByte(edf)

		// Write the EDF data.
		buf.Write(edfBuf.Bytes())
	}

	// Write the packet out.
	_, err := io.Copy(out, buf)
	return err
}

func (t *TestServer) writePlayerInfo(out io.Writer) error {
	info := t.PlayerInfo
	if info == nil {
		info = &a2s.PlayerInfo{}
	}

	// Response packet buffer.
	buf := &packetBuffer{}

	// Header.
	buf.WriteUInt32(math.MaxUint32)
	buf.WriteByte('D')

	// Payload.
	buf.WriteByte(info.Count)

	for _, player := range info.Players {
		buf.WriteByte(player.Index)
		buf.WriteCString(player.Name)
		buf.WriteUInt32(player.Score)
		buf.WriteFloat32(player.Duration)

		if t.ServerInfo != nil && a2s.AppID(t.ServerInfo.ID) == a2s.App_TheShip {
			theShip := player.TheShip
			if theShip == nil {
				theShip = &a2s.TheShipPlayer{}
			}
			buf.WriteUInt32(theShip.Deaths)
			buf.WriteUInt32(theShip.Money)
		}
	}

	// Write the packet out.
	_, err := io.Copy(out, buf)
	return err
}

// packetBuffer extends bytes.Buffer to add more data types used by A2S.
type packetBuffer struct {
	bytes.Buffer
}

func (b *packetBuffer) WriteCString(s string) {
	b.WriteString(s)
	b.WriteByte(0)
}

func (b *packetBuffer) WriteUInt16(v uint16) {
	var buf [2]byte
	binary.LittleEndian.PutUint16(buf[:], v)
	b.Write(buf[:])
}

func (b *packetBuffer) WriteUInt32(v uint32) {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], v)
	b.Write(buf[:])
}

func (b *packetBuffer) WriteUInt64(v uint64) {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], v)
	b.Write(buf[:])
}

func (b *packetBuffer) WriteBool(v bool) {
	var c byte
	if v {
		c = 1
	}
	b.WriteByte(c)
}

func (b *packetBuffer) WriteFloat32(v float32) {
	b.WriteUInt32(math.Float32bits(v))
}

// udpWriter writes UDP packets to an address.
type udpWriter struct {
	conn net.PacketConn
	addr net.Addr
}

func (w *udpWriter) Write(p []byte) (int, error) {
	return w.conn.WriteTo(p, w.addr)
}

func formatServerType(serverType a2s.ServerType) byte {
	switch serverType {
	case a2s.ServerType_Dedicated:
		return 'd'
	case a2s.ServerType_NonDedicated:
		return 'l'
	case a2s.ServerType_SourceTV:
		return 'p'
	default:
		return 0
	}
}

func formatServerOS(serverOS a2s.ServerOS) byte {
	switch serverOS {
	case a2s.ServerOS_Linux:
		return 'l'
	case a2s.ServerOS_Windows:
		return 'w'
	case a2s.ServerOS_Mac:
		return 'm'
	default:
		return 0
	}
}

func formatTheShipMode(mode a2s.TheShipMode) byte {
	switch mode {
	case a2s.TheShipMode_Hunt:
		return 0
	case a2s.TheShipMode_Elimination:
		return 1
	case a2s.TheShipMode_Duel:
		return 2
	case a2s.TheShipMode_Deathmatch:
		return 3
	case a2s.TheShipMode_VIP_Team:
		return 4
	case a2s.TheShipMode_Team_Elimination:
		return 5
	default:
		return 0
	}
}

package warm

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

func pingMinecraft(addr string) error {
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(5 * time.Second))

	host, portStr, _ := net.SplitHostPort(addr)
	var port uint16
	fmt.Sscanf(portStr, "%d", &port)

	if err := sendHandshake(conn, host, port); err != nil {
		return fmt.Errorf("sending handshake: %w", err)
	}

	if err := sendStatusRequest(conn); err != nil {
		return fmt.Errorf("sending status request: %w", err)
	}

	if err := readStatusResponse(conn); err != nil {
		return fmt.Errorf("reading status response: %w", err)
	}

	return nil
}

func sendHandshake(conn net.Conn, host string, port uint16) error {
	var payload bytes.Buffer

	writeVarInt(&payload, 0x00)
	writeVarInt(&payload, 774)
	writeString(&payload, host)
	binary.Write(&payload, binary.BigEndian, port)
	writeVarInt(&payload, 1)

	return writePacket(conn, payload.Bytes())
}

func sendStatusRequest(conn net.Conn) error {
	var payload bytes.Buffer
	writeVarInt(&payload, 0x00)
	return writePacket(conn, payload.Bytes())
}

func readStatusResponse(conn net.Conn) error {
	_, err := readVarInt(conn)
	if err != nil {
		return fmt.Errorf("reading packet length: %w", err)
	}

	packetID, err := readVarInt(conn)
	if err != nil {
		return fmt.Errorf("reading packet id: %w", err)
	}

	if packetID != 0x00 {
		return fmt.Errorf("unexpected packet id: %d", packetID)
	}

	return nil
}

func writePacket(conn net.Conn, data []byte) error {
	var buf bytes.Buffer
	writeVarInt(&buf, int32(len(data)))
	buf.Write(data)
	_, err := conn.Write(buf.Bytes())
	return err
}

func writeVarInt(buf *bytes.Buffer, value int32) {
	uval := uint32(value)
	for {
		if uval&^0x7F == 0 {
			buf.WriteByte(byte(uval))
			return
		}
		buf.WriteByte(byte(uval&0x7F) | 0x80)
		uval >>= 7
	}
}

func writeString(buf *bytes.Buffer, s string) {
	writeVarInt(buf, int32(len(s)))
	buf.WriteString(s)
}

func readVarInt(conn net.Conn) (int32, error) {
	var result int32
	var shift uint
	buf := make([]byte, 1)

	for {
		if _, err := conn.Read(buf); err != nil {
			return 0, err
		}
		result |= int32(buf[0]&0x7F) << shift
		if buf[0]&0x80 == 0 {
			return result, nil
		}
		shift += 7
		if shift >= 35 {
			return 0, fmt.Errorf("VarInt too big")
		}
	}
}

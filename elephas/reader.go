package elephas

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
)

type Reader struct {
	*bufio.Reader
}

type DataRow struct {
}

func NewReader(r *bufio.Reader) *Reader {
	return &Reader{r}
}

func (r Reader) ReadBytesToUint32(size uint) (uint32, error) {
	b := make([]byte, size)
	_, err := io.ReadFull(r, b)
	return binary.BigEndian.Uint32(b), err
}

func (r Reader) ReadBytesToAny(size uint32, dataType int) (any, error) {
	b := make([]byte, size)
	_, err := io.ReadFull(r, b)
	if err != nil {
		return nil, err
	}
	switch dataType {
	case 23:
		v, err := strconv.Atoi(string(b))
		if err != nil {
			return nil, nil
		}
		return v, nil
	case 25:
		return string(b), nil
	case 16:
		return strconv.ParseBool(string(b))
	default:
		panic(fmt.Sprintf("the OID type %v is not implemented", dataType))
	}
}

func (r Reader) ReadBytesToUint16(size uint) (uint16, error) {
	b := make([]byte, size)
	_, err := io.ReadFull(r, b)
	return binary.BigEndian.Uint16(b), err
}

func (r Reader) handleAuthResp(authType uint32) ([]byte, error) {
	if t, err := r.Reader.ReadByte(); err != nil {
		return nil, err
	} else if t != authMsgType {
		return nil, fmt.Errorf("expect message type is authentication (%v) but got: %v", authMsgType, t)
	}
	l, err := r.ReadBytesToUint32(4)
	l -= 8 //
	if err != nil {
		return nil, err
	}
	respAuthType, err := r.ReadBytesToUint32(4)
	if respAuthType != authType {
		return nil, fmt.Errorf("expect authentication type (%v) but got: %v", authType, respAuthType)
	}
	if l == 0 { // the end of the response
		return nil, nil
	}
	// i like those letters 't', 'l'. They confuse the reader, haha
	d := make([]byte, l)
	if _, err := io.ReadFull(r.Reader, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (r Reader) readRowDescription(conn net.Conn) (Rows, error) {
	msgType, err := r.ReadByte()
	if err != nil {
		return Rows{}, err
	}
	if msgType != rowDescription {
		return Rows{}, fmt.Errorf("Expect Row Description type but got %v", msgType)
	}
	msgLen, err := r.ReadBytesToUint32(4)
	if err != nil {
		return Rows{}, errors.New("readRowDescription: Failed to read msgLen")
	}
	fieldCount, err := r.ReadBytesToUint16(2)
	if err != nil {
		return Rows{}, errors.New("readRowDescription: Failed to read fieldCount")
	}
	log.Println(msgLen, fieldCount)
	var rows Rows
	for i := 0; i < int(fieldCount); i++ {
		fieldName, err := r.ReadString(0)
		if err != nil {
			return Rows{}, errors.New("readRowDescription: Failed to read fieldName")
		}
		rows.cols = append(rows.cols, fieldName)
		r.Discard(4 + 2)
		oid, err := r.ReadBytesToUint32(4)
		if err != nil {
			return Rows{}, errors.New("readRowDescription: Failed to read oid")
		}
		rows.oids = append(rows.oids, int(oid))
		r.Discard(2 + 4 + 2)
	}
	rows.reader = &r
	rows.conn = conn
	return rows, nil
}

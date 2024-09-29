package elephas

import (
	"bufio"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"mellium.im/sasl"
)

// Conn
type Connection struct {
	cfg    *Config
	conn   net.Conn
	reader *Reader
}

func (c *Connection) Prepare(query string) (driver.Stmt, error) {
	panic("not implemented")
}

func (c *Connection) Close() error {
	return c.conn.Close()
}

// deprecated function, use BeginTx instead
func (c *Connection) Begin() (driver.Tx, error) {
	panic("not implemented")
}

func (c *Connection) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if sql.IsolationLevel(opts.Isolation) != sql.LevelDefault {
		return nil, errors.New("not implemented")
	}
	if opts.ReadOnly {
		return nil, errors.New("not implemented")
	}
	var b Buffer
	_, err := c.conn.Write(b.writeQuery("Begin", nil))
	if err != nil {
		return nil, err
	}
	cmdTag, err := c.reader.ReadBeginTxResponse()
	if err != nil {
		return nil, fmt.Errorf("unable to ReadAndExpect(%v)", commandComlete)
	}
	if cmdTag != "BEGIN" {
		return nil, fmt.Errorf("expect BEGIN command tag but got (%v)", cmdTag)
	}
	txStatus, err := c.reader.ReadReadyForQuery()
	if err != nil {
		return nil, fmt.Errorf("unable to ReadAndExpect(%v)", txStatus)
	}
	if txStatus == T {
		log.Println("in tx")
	}

	return NewTransaction(), nil
}

// https://www.postgresql.org/docs/current/protocol-flow.html#PROTOCOL-FLOW-START-UP
func (c *Connection) makeHandShake() error {
	var b Buffer
	if _, err := c.conn.Write(b.buildStartUpMsg()); err != nil {
		log.Fatalf("Failed to make hande shake: %v", err)
		return err
	}
	log.Println("Sent StartupMessage")
	for {
		msgType, err := c.reader.ReadByte()
		if err != nil {
			return err
		}
		msgLen, err := c.reader.ReadBytesToUint32(4)
		if err != nil {
			return err
		}
		switch msgType {
		case authMsgType:
			if err = c.doAuthentication(b); err != nil {
				log.Printf("Failed to do authentication: %v", err)
				return err
			}
		case parameterStatus:
			// https://www.postgresql.org/docs/current/protocol-flow.html#PROTOCOL-ASYNC
			c.reader.Discard(int(msgLen - 4))
		case backendKeyData:
			c.reader.Discard(int(msgLen - 4))
		case readyForQuery:
			c.reader.Discard(int(msgLen - 4))
			return nil
		default:
			log.Println(string(msgType))
		}
	}
}

func (c *Connection) doAuthentication(b Buffer) error {
	authType, err := c.reader.ReadBytesToUint32(4)
	if err != nil {
		return err
	}
	if authType == SASL {
		data, err := c.reader.ReadBytes(0)
		c.reader.ReadByte() // get  rid of last byte
		if err != nil {
			log.Fatalf("Failed to handle AuthenticationSASL; %v", err)
			return err
		}
		switch string(string(data[:len(data)-1])) {
		case sasl.ScramSha256.Name:
			log.Println("Start SASL authentication")
			err = c.authSASL(&b)
			if err != nil {
				return err
			}
			log.Println("Finish SASL authentication")
		default:
			panic("TODO ScramSha256Plus")
		}
	} else {
		panic("TODO SASL")
	}
	return nil
}

func (c *Connection) authSASL(b *Buffer) error {
	creds := sasl.Credentials(func() (Username []byte, Password []byte, Identity []byte) {
		return []byte(c.cfg.User), []byte(c.cfg.Password), []byte{}
	})
	client := sasl.NewClient(sasl.ScramSha256, creds)
	_, resp, err := client.Step(nil) // n,,n=postgres,r= nonce
	if err != nil {
		log.Printf("Failed to Step: %v \n", err)
		return err
	}
	_, err = c.conn.Write(b.buildSASLInitialResponse(resp))
	if err != nil {
		log.Printf("Failed to send SASLInitialResponse: %v", err)
		return err
	}
	data, err := c.reader.handleAuthResp(SASLContinue)
	if err != nil {
		log.Println("Failed to handle AuthenticationSASLContinue")
		return err
	}
	_, resp, err = client.Step(data)
	if err != nil {
		log.Printf("Failed to step: %v \n", err)
		return err
	}

	_, err = c.conn.Write(b.buildSASLResponse(resp))
	if err != nil {
		log.Printf("Failed to send SASLResponse (Step4): %v \n", err)
		return err
	}

	data, err = c.reader.handleAuthResp(SASLComplete)
	if err != nil {
		log.Printf("Failed to handle AuthenticationSASLFinal (complete): %v", err)
		return err
	}
	if _, _, err = client.Step(data); err != nil {
		log.Printf("client.Step 3 failed: %v", err)
		return err
	}
	if client.State() != sasl.ValidServerResponse {
		log.Printf("invalid server reponse: %v", client.State())
	}
	_, err = c.reader.handleAuthResp(AuthSuccess)
	if err != nil {
		log.Printf("Failed to handle AuthenticationSASLFinal (success): %v", err)
		return err
	}
	return nil
}

func NewConnection(ctx context.Context, cfg *Config) (*Connection, error) {
	d := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 5 * time.Minute,
	}

	dConn, err := d.DialContext(ctx, cfg.Network, cfg.Addr)
	if err != nil {
		log.Fatalf("Failed to dial: %v", err)
		return nil, err
	}
	reader := NewReader(bufio.NewReader(dConn))
	conn := Connection{conn: dConn, reader: reader, cfg: cfg}
	if err := conn.makeHandShake(); err != nil {
		log.Fatalf("Failed to make handle shake: %v", err)
		return nil, err
	}

	return &conn, nil
}

func (c *Connection) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	var b Buffer
	_, err := c.conn.Write(b.writeQuery(query, args))
	if err != nil {
		log.Printf("Failed to send Query: %v", err)
		return nil, err
	}
	rows, err := c.reader.readRowDescription(c.conn)
	if err != nil {
		return nil, err
	}
	return (&rows), nil
}

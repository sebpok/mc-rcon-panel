package rcon

import "github.com/gorcon/rcon"

type Client struct {
	conn *rcon.Conn
}

func Connect(addr string, password string) (*Client, error) {
	conn, err := rcon.Dial(addr, password)
	if err != nil {
		return nil, err
	}

	return &Client{conn: conn}, nil
}

func (c *Client) Exec(cmd string) (string, error) {
	return c.conn.Execute(cmd)
}

func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

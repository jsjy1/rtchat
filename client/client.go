package client

import (
	"fmt"
	"io"
	"net"
)

type Client struct {
	Conn net.Conn
	Name string //初始命名为ip:port
}

func NewClient(ip string, port int) *Client {

	ip_port := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.Dial("tcp", ip_port)
	if err != nil {
		println("dial err: ", err.Error())
		return nil
	}
	c := &Client{
		Conn: conn,
		Name: ip_port,
	}
	return c
}

func (c *Client) Close() {
	c.Conn.Close()
}

func (c *Client) Send(s string) int {
	n, err := c.Conn.Write([]byte(s + "\n"))
	if err != nil {
		println("write err: ", err.Error())
	}
	return n
}

func (c *Client) Recv() (string, int) {
	buf := make([]byte, 4096)
	n, err := c.Conn.Read(buf)
	if n == 0 {
		// 连接关闭有3种情况，一种自己quit，一种自己超时服务器器踢出，一种服务器断开
		// println("server is close...")
		c.Close()
		return "", 0
	}
	if err != nil && err != io.EOF {
		println("read err: ", err.Error())
		return "", -1
	}
	return string(buf[:n]), n
}

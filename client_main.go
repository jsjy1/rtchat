package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"rtchat/client"
)

var server_ip string
var server_port int

func init() {
	flag.StringVar(&server_ip, "ip", "127.0.0.1", "server ip, default 127.0.0.1.")
	flag.IntVar(&server_port, "port", 8888, "server port, default 8888.")

	// 命令行解析
	flag.Parse()
}

func main() {
	// 创建连接
	c := client.NewClient(server_ip, server_port)
	if c == nil {
		println("connect server failed!")
		return
	}
	defer c.Close()
	println("connect server success!")

	close := make(chan bool)

	// 监听输入
	go inputEvent(c, close)

	// 监听server消息
	go serverEvent(c, close)

	// 监听关闭
	select {
	case <-close:
	}
}

func inputEvent(c *client.Client, close chan bool) {
	reader := bufio.NewReader(os.Stdin)
	for {
		instr, err := reader.ReadString('\n')
		if err != nil {
			println("input get err: ", err.Error())
			continue
		}
		instr = instr[:len(instr)-1]
		// 退出
		if instr == "\\q" {
			close <- true
			return
		}
		c.Send(instr)
	}
}

func serverEvent(c *client.Client, close chan bool) {
	for {
		msg, n := c.Recv()
		// 收到close则关闭
		if n == 0 {
			close <- true
			return
		}
		//输出消息
		fmt.Printf("%s", msg)
	}
}

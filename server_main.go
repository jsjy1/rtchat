package main

import (
	"rtchat/server"
)

func main() {
	s := server.NewServer("127.0.0.1", 8888) // 这个不能是成员方法，否则好像不能写成static通过类直接调用
	s.Start()
}

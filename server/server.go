package server

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

type Server struct {
	Ip   string
	Port int

	usrMap map[string]*User
	lck    sync.RWMutex

	infoqueue  chan string // 用于接收info chan自带同步功能
	closequeue chan string // 用于接收关闭client 准备从连接表中删除
}

func NewServer(ip string, port int) *Server { // 这个不能是成员方法
	server := &Server{ // 相当于new了一个对象
		Ip:         ip,
		Port:       port,
		usrMap:     make(map[string]*User),
		lck:        sync.RWMutex{},
		infoqueue:  make(chan string, 5),
		closequeue: make(chan string, 5),
	}
	return server
}

/*
	 server
								创建socket
		合成一步了listen          bind ip port
								listen
		accept
*/
func (s *Server) Start() {
	// tcp流程
	lfd, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.Ip, s.Port))
	if err != nil {
		println("listen error: ", err.Error())
		return
	}

	defer func() {
		lfd.Close()
		close(s.infoqueue)
		close(s.closequeue)
	}()

	// 监听chan
	go s.listen_chan()

	for {
		// accept建立连接
		cfd, err := lfd.Accept()
		if err != nil {
			println("listen error: ", err.Error())
			return
		}

		// go程 处理事件
		go s.handle_conn(cfd)
	}
}

func (s *Server) handle_conn(cfd net.Conn) {
	// ser端输出
	println("new connect!", cfd.RemoteAddr().String())

	// 创建对应usr并加入map
	usr := NewUser(cfd, s.infoqueue, s.closequeue)

	s.lck.Lock()
	s.usrMap[usr.Name] = usr
	s.lck.Unlock()

	// cli端输出
	s.infoqueue <- fmt.Sprintf("%s 加入房间.", usr.Name)
}

/*
命令检测
return

	   -1  该info错误
		0, _, _   无命令
		1, name, command 有命令，发出者，命令
*/
func (s *Server) command_check(info *string) (int, string, string) {
	if len(*info) <= 1 {
		println("command check error: len(info) <=1.")
		return -1, "", ""
	}
	if (*info)[0] != ' ' { // 没有命令
		return 0, "", ""
	}

	// 命令格式: [ name 命令]
	j := 1
	for {
		if (*info)[j] == ' ' || j >= len(*info) {
			break
		}
		j++
	}
	return 1, (*info)[1:j], (*info)[j+1:]
}

/*
执行命令(包括命令检测) 目前所有命令全放在这里面
TODO: 之后可以命令分开实现，用枚举量表示
\help
\who
\rename
\toff
*/
func (s *Server) do_command(name, com *string) {
	tmp := strings.Split(*com, " ")
	s.lck.Lock()

	switch tmp[0] {
	case "\\help":
		if len(tmp) > 1 {
			s.usrMap[*name].Send("do you want to use [\\help]?\n")
			break
		}
		// 给name发info
		sendmsg := "  [\\help]\t\tGet available command;\n" +
			"  [\\who]\t\tFind all online users;\n" +
			"  [\\rename new_name]\tRename;\n" +
			"  [\\to user info]\tPrivate chat user;\n" +
			"\n"
		s.usrMap[*name].Send(sendmsg)
	case "\\who":
		if len(tmp) > 1 {
			s.usrMap[*name].Send("do you want to use [\\who]?\n")
			break
		}
		sendmsg := ""
		for k := range s.usrMap {
			sendmsg += "[" + k + "]" + "\n"
		}
		sendmsg += "\n"
		s.usrMap[*name].Send(sendmsg)
	case "\\rename":
		if len(tmp) == 1 {
			s.usrMap[*name].Send("do you want to use [\\rename new_name]?\n")
			break
		}
		// 名字是否合法 是否有名字存在
		if len(tmp) > 2 || len(tmp[1]) <= 1 {
			s.usrMap[*name].Send("New name can't contain Spaces and its length must be at least 2.\n")
			break
		}

		_, ok := s.usrMap[tmp[1]]
		if ok {
			s.usrMap[*name].Send("This name exists, rename fail.\n")
			break
		}

		// 更改名字  server usr
		s.usrMap[tmp[1]] = s.usrMap[*name]
		delete(s.usrMap, *name)
		s.usrMap[tmp[1]].Name = tmp[1]
	case "\\to":
		if len(tmp) < 3 {
			s.usrMap[*name].Send("do you want to use [\\to user info]?\n")
			break
		}
		// 合法性及usr是否存在
		if tmp[1] == *name {
			s.usrMap[*name].Send("You can't speak to youself.\n")
			break
		}
		_, ok := s.usrMap[tmp[1]]
		if !ok {
			s.usrMap[*name].Send("[" + tmp[1] + "]" + " offline.\n")
			break
		}
		sendmsg := (*com)[5+len(tmp[1]):] // 提取info
		sendmsg = fmt.Sprintf("[%s %s speak to you] %s \n\n", time.Now().String()[:19], *name, sendmsg)
		s.usrMap[tmp[1]].Send(sendmsg)

	default:
		s.usrMap[*name].Send("unkonw command, you can use \\help to get available command.\n")
	}

	s.lck.Unlock()
}

/*
循环检测是否有info，一旦有info就广播给所有人
使用for循环实现
*/
func (s *Server) listen_chan() {
	for {
		select {
		// 接收到info
		case sendmsg := <-s.infoqueue:
			// 解析info
			ok, name, command := s.command_check(&sendmsg)

			switch ok {
			case -1: // 解析错误
			case 0: // 正常语句
				s.lck.Lock()
				for _, v := range s.usrMap {
					tmp := fmt.Sprintf("[%s] %s\n\n", time.Now().String()[:19], sendmsg)
					v.Send(tmp)
				}
				s.lck.Unlock()
			case 1: //命令
				s.do_command(&name, &command)
			default:
				println("unknow command check...")
			}

		// close
		case closemsg := <-s.closequeue:
			// 删除对应连接 并广播通知
			s.lck.Lock()
			delete(s.usrMap, closemsg)
			println(closemsg, " quit.")
			sendmsg := fmt.Sprintf("%s 离开了...", closemsg)
			for _, v := range s.usrMap {
				tmp := fmt.Sprintf("[%s] %s\n", time.Now().String()[:19], sendmsg)
				v.Send(tmp)
			}
			s.lck.Unlock()
		}
	}
}

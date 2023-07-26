package server

import (
	"io"
	"net"
	"sync"
	"time"
)

const overtime time.Duration = 10 // 超时时间

/*
负责连接关闭，数据传输
只负责执行操作
*/
type User struct {
	Name string
	cfd  net.Conn

	tim *time.Timer // 记录未活跃时间
	lck sync.Mutex  // 维护定时器
}

/*
创建一个user，同时需要开始监听cfd是否有数据，有则放到ser的chan内
*/
func NewUser(cfd net.Conn, infoqueue, closequeue chan string) *User {
	usr := &User{
		Name: cfd.RemoteAddr().String(),
		cfd:  cfd,
		tim:  time.NewTimer(overtime * time.Second),
		lck:  sync.Mutex{},
	}

	go usr.Recv(infoqueue, closequeue) // 同时监听是否有数据到达
	go usr.overtime_check(closequeue)  // 超时检测

	return usr
}

/*
	 usr发送字符串
		return
			-1 : error
			>0 : success
*/
func (u *User) Send(s string) int {
	n, err := u.cfd.Write([]byte(s))
	if err != nil {
		println("write error: ", err.Error())
		return -1
	}
	return n
}

/*
usr在go程里read的，如果对面关闭了，那怎么通知server删除该连接呢 TODO:
专门利用一个chan来通知下线？ 其实这里分离recv和循环判断比较好

	return

		0  close
		-1 err
		>0 success	实际上会循环rcvd，利用chan传出info
*/
func (u *User) Recv(infoqueue, closequeue chan string) int {
	buf := make([]byte, 1000)

	// 循环检测
	for {
		n, err := u.cfd.Read(buf)
		if n == 0 { // 用户下线 需要先判断下线 再判断其他错误，因为对方关闭只是错误的一种
			u.close(closequeue)
			return 0
		}
		if err != nil && err != io.EOF { // TODO:
			println("read error: ", err.Error())
			return -1
		}

		// 命令检测后 info传递给schan
		rcvmsg := string(buf)[:n-1] // 去除最后\n
		ok := u.command_check(&rcvmsg)
		if ok {
			rcvmsg = " " + u.Name + " " + rcvmsg
		} else {
			rcvmsg = u.Name + ": " + rcvmsg
		}
		println(rcvmsg)
		infoqueue <- rcvmsg

		// 重置定时器
		u.lck.Lock()
		u.tim.Reset(overtime * time.Second)
		u.lck.Unlock()
	}
}

/*
命令检测
return

	0 没有命令
	1 有命令
*/
func (u *User) command_check(s *string) bool {
	if len(*s) == 0 || (*s)[0] != '\\' { // 长度为0 或者 不是命令
		return false
	}
	return true
}

/*
超时检测
TODO: 如果定时器停止了，那这个会一直阻塞在这，go程无法停止
*/
func (u *User) overtime_check(closequeue chan string) {
	for {
		select {
		case <-u.tim.C: // 时间到时，curtime会被发送到C中
			u.Send("太长时间未发送消息, 已被踢出...\n")
			u.close(closequeue)
			return
		}
	}
}

/*
关闭连接 并通知server 关闭定时器
*/
func (u *User) close(closequeue chan string) {
	u.cfd.Close()
	closequeue <- u.Name
	u.tim.Stop()
}

package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"time"
)

type User struct {
	name   string
	conn   *websocket.Conn
	C      chan string
	server *Server
}

//NewUser 新建用户
func NewUser(name string, ws *websocket.Conn, server *Server) *User {
	user := &User{
		name:   name,
		conn:   ws,
		C:      make(chan string),
		server: server,
	}
	go user.listenChan()
	return user
}

func (this *User) listenChan() {
	for {
		msg := <-this.C
		if len(msg) > 0 {
			err := this.conn.WriteMessage(websocket.TextMessage, []byte(msg))
			if err != nil {
				return
			}
		}
	}
}

func (this *User) Send(Msg string) {
	this.C <- Msg
}

func (this *User) OnMessage(messageArr []string) {

	switch messageArr[0] {
	case "list":
		//>> 获取列表
		arr := ""
		this.server.UserMapLock.RLock()
		for _, user := range this.server.UserMap {
			arr = arr + user.name + "|"
		}
		this.server.UserMapLock.RUnlock()
		this.server.UserMap[messageArr[1]].Send("list|-|" + arr)
		break
	case "send":
		//>> 发送消息 1 username 2 time 3 msg
		if len(messageArr[2]) > 0 {
			t := time.Now()
			tStr := fmt.Sprintf("%02d:%02d:%02d", t.Hour(), t.Minute(), t.Second())
			this.server.Broadcast("msg|-|" + messageArr[1] + "|-|" + tStr + "|-|" + messageArr[2] + "")
		}

		break
	}
}

func (this *User) Online() {
	// 锁住
	this.server.UserMapLock.Lock()
	// 创建到map
	this.server.UserMap[this.name] = this
	this.server.UserMapLock.Unlock()
	//>> 1 状态  2 用户名  3 信息
	this.server.Broadcast("tip|-|1|-|" + this.name + "|-|欢迎[" + this.name + "]上线")
}

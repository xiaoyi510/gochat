package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"time"
)

type User struct {
	name       string
	conn       *websocket.Conn
	C          chan string
	server     *Server
	lastAcTime int64
}

//NewUser 新建用户
func NewUser(name string, ws *websocket.Conn, server *Server) *User {
	user := &User{
		name:       name,
		conn:       ws,
		C:          make(chan string),
		server:     server,
		lastAcTime: time.Now().Unix(),
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
		this.onList()
		break
	case "set-username":
		this.onSetUsername(messageArr[2])
		break
	case "send":
		//>> 发送消息 1 username 2 time 3 msg
		if len(messageArr[2]) > 0 {
			this.onSend(messageArr[1], messageArr[2])
		}
		break
	}
}

func (this *User) onList() {
	//>> 获取用户列表
	arr := ""
	this.server.UserMapLock.RLock()
	for _, user := range this.server.UserMap {
		arr = arr + user.name + "|"
	}
	this.server.UserMapLock.RUnlock()
	this.server.UserMap[this.name].Send("list|-|" + arr)
}

// 发送消息
func (this *User) onSend(username string, msg string) {
	t := time.Now()
	tStr := fmt.Sprintf("%02d:%02d:%02d", t.Hour(), t.Minute(), t.Second())
	this.server.Broadcast("msg|-|" + username + "|-|" + tStr + "|-|" + msg + "")
}

func (this *User) Online() {
	// 锁住
	this.server.UserMapLock.Lock()
	// 创建到map
	this.server.UserMap[this.name] = this
	this.server.UserMapLock.Unlock()
	//>> 1 状态  2 用户名  3 信息
	this.server.Broadcast(this.getTipText(TIP_TYPE_USER_ONLINE, "欢迎["+this.name+"]加入聊天室"))

	go this.checkStatus()
}

// 检测用户状态
func (this *User) checkStatus() {

	for {
		if this.lastAcTime >= time.Now().Unix()-60 {
			time.Sleep(time.Second * 1)
		} else {
			err := this.conn.Close()
			if err != nil {
				return
			}
		}
	}
}

// 修改用户名
func (this *User) onSetUsername(newUsername string) {
	// 删除关系

	// 修改到用户映射关系中
	if _, ok := this.server.UserMap[newUsername]; ok {
		//>> 已有用户存在
		this.SendTip(TIP_TYPE_ERROR_NORMAL, "用户名["+newUsername+"]已存在")
		return
	}
	// 记录老用户名
	oldName := this.name

	this.server.UserMapLock.Lock()
	this.server.UserMap[newUsername] = this
	this.server.UserMapLock.Unlock()
	delete(this.server.UserMap, oldName)

	// 修改为新用户名
	this.name = newUsername

	// 通知客户端用户已修改用户名
	this.topSetUsernameTip(oldName)
}

// 发送用户修改用户名成功
func (this *User) topSetUsernameTip(oldUsername string) {
	this.server.Broadcast(this.getTipText(TIP_TYPE_USER_SET_USERNAME, oldUsername))

}

// SendTip 发送公告
func (this *User) SendTip(tipType string, msg string) {
	this.Send(this.getTipText(tipType, msg))
}

func (this *User) getTipText(tipType string, msg string) string {
	return "tip|-|" + tipType + "|-|" + this.name + "|-|" + msg
}

// OffLine 用户下线
func (this *User) OffLine() {
	delete(this.server.UserMap, this.name)
	this.server.Broadcast(this.getTipText(TIP_TYPE_USER_OFFLINE, "用户["+this.name+"]离开了"))
	err := this.conn.Close()
	if err != nil {
		return
	}
}

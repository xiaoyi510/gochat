package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Server struct {
	Ip          string
	Port        int
	UserMap     map[string]*User
	UserMapLock sync.RWMutex
	Msg         chan string
}

// NewServer 创建服务器
func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip:      ip,
		Port:    port,
		UserMap: make(map[string]*User),
		Msg:     make(chan string),
	}
	return server
}

// 升级到websocket
var upGrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Start 开始服务
func (this *Server) Start() {
	r := gin.Default()
	go this.listenMsgChan()

	// 指明html加载文件目录
	r.LoadHTMLGlob("./client_html/*")
	r.Handle("GET", "/", func(context *gin.Context) {
		// 返回HTML文件，响应状态码200，html文件名为index.html，模板参数为nil
		context.HTML(http.StatusOK, "index.html", nil)
	})

	// 监听Get请求
	r.GET("/websocket", this.handle)

	r.Run(fmt.Sprintf("%s:%d", this.Ip, this.Port))

}

// 处理用户进入
func (this *Server) handle(c *gin.Context) {
	println("有用户进入")
	//升级get请求为webSocket协议
	ws, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	// 当前用户信息
	var user *User

	// 端口连接后处理
	defer func() {
		if user != nil && len(user.name) > 0 {
			user.OffLine()
		} else {
			err := ws.Close()
			if err != nil {
				return
			}
		}
	}()

	for {
		//读取ws中的数据
		_, message, err := ws.ReadMessage()
		if err != nil {
			break
		}
		if string(message) == "ping" {
			message = []byte("pong")
		}
		messageArr := strings.Split(string(message), "|_|")

		//判断数据是否合法
		if len(messageArr) == 0 {
			continue
		}

		if user != nil {
			user.lastAcTime = time.Now().Unix()
		}

		switch messageArr[0] {
		case "init":
			if _, ok := this.UserMap[messageArr[1]]; ok {
				ws.WriteMessage(websocket.TextMessage, []byte("tip|-|"+TIP_TYPE_ERROR+"|-|"+messageArr[1]+"|-|用户已登录请切换其他账号"))
				return
			}
			// 创建用户
			user = NewUser(messageArr[1], ws, this)
			// 处理用户上线
			user.Online()
			// 记录当前协程用户名

			break
		default:
			if user != nil {
				user.OnMessage(messageArr)
			} else {
				ws.WriteMessage(websocket.TextMessage, []byte("tip|-|"+TIP_TYPE_ERROR+"|-|1|-|请先登录"))
				return
			}
		}

	}

}

// Broadcast 群发消息
func (this *Server) Broadcast(msg string) {
	this.Msg <- msg
}

// 群发消息监听
func (this *Server) listenMsgChan() {
	for {
		msg := <-this.Msg
		// 如果消息内容不为空则群发消息
		if len(msg) > 0 {
			//>> 开始广播
			this.UserMapLock.RLock()
			for _, user := range this.UserMap {
				user.Send(msg)
			}
			this.UserMapLock.RUnlock()
		}
	}
}

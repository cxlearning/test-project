package websocket

import (
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"

	"github.com/gin-gonic/gin"
	uuid "github.com/satori/go.uuid"
)

//Client:单个websocket
type Client struct {
	Id      string
	Socket  *websocket.Conn
	Message chan []byte
}

var clientCount uint // 客户端数量

//从websocket中直接读取数据
func (c *Client) Read() {
	defer func() {
		//客户端关闭
		if err := c.Socket.Close(); err != nil {
			fmt.Printf("client [%s] disconnect err: %s", c.Id, err)
		}
		//关闭后直接注销客户端
		//WebsocketManager.UnRegisterClient(c)
		clientCount--
		fmt.Printf("client [%s],客户端关闭：[%s],Count [%d]\n", c.Id, websocket.CloseMessage, clientCount)
	}()

	for {
		messageType, message, err := c.Socket.ReadMessage()
		//读取数据失败
		if err != nil || messageType == websocket.CloseMessage {
			fmt.Printf("client [%s],数据读取失败或通道关闭：[%s],客户端连接状态：[%s]\n", c.Id, err.Error(), websocket.CloseMessage)
			break
		}

		// TODO 解析发送过来的参数
		//var data ReadData
		//err = json.Unmarshal(message, &data)
		//if err != nil {
		//  fmt.Println("数据解析失败")
		//	return
		//}

		// TODO 前端请求返回数据到指定客户端

		// 简单测试
		c.Message <- message

	}
}

//写入数据到websocket中
func (c *Client) Write() {
	defer func() {
		//客户端关闭
		if err := c.Socket.Close(); err != nil {
			fmt.Printf("client [%s] disconnect err: %s \n", c.Id, err)
			return
		}
		//关闭后直接注销客户端
		//WebsocketManager.UnRegisterClient(c)
		clientCount--
		fmt.Printf("client [%s],客户端关闭：[%s]\n", c.Id, websocket.CloseMessage)
	}()

	for {
		select {
		case message, ok := <-c.Message:

			if !ok {
				//数据写入失败，关闭通道
				fmt.Printf("client [%s],客户端连接状态：[%s]\n", c.Id, websocket.CloseMessage)
				_ = c.Socket.WriteMessage(websocket.CloseMessage, []byte{})
				//消息通道关闭后直接注销客户端
				return
			}

			err := c.Socket.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				fmt.Printf("client [%s] write message err: %s \n", c.Id, err)
				return
			}
		}
	}
}

// 方法二: 通过对象创建  客户端连接
func WsClient(context *gin.Context) {
	upGrande := websocket.Upgrader{
		//设置允许跨域
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		//设置请求协议
		Subprotocols: []string{context.GetHeader("Sec-WebSocket-Protocol")},
	}
	//创建连接
	conn, err := upGrande.Upgrade(context.Writer, context.Request, nil)
	if err != nil {
		fmt.Printf("websocket connect error: %s", context.Param("channel"))
		//format.NewResponseJson(context).Error(51001)
		return
	}
	//生成唯一标识client_id
	var uuid = uuid.NewV4().String()
	client := &Client{
		Id:      uuid,
		Socket:  conn,
		Message: make(chan []byte, 1024),
	}
	//注册
	//ws.WebsocketManager.RegisterClient(client)
	clientCount++

	//起协程，实时接收和回复数据
	go client.Read()
	go client.Write()
}

// 方法一: 直接创建客户端
func NewConnection(c *gin.Context) {
	// 定义同源检查，这里只作简单试验不校验
	upGrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	ws, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	//ws, err := websocket.Upgrade(c.Writer, c.Request, nil, 1024, 1024)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"msg": "服务端错误",
		})
		return
	}

	var message = make(chan []byte)

	go func() {
		defer ws.Close()

		for {
			fmt.Println("start transfer message")
			msgType, msg, err := ws.ReadMessage()
			if err != nil || websocket.CloseMessage == msgType {
				fmt.Println("webSocket read error")
				return
			}
			message <- msg
		}

	}()

	go func() {
		defer ws.Close()

		for {
			mm, ok := <-message

			if !ok {
				//数据写入失败，关闭通道
				fmt.Printf("客户端连接状态：[%s]\n", websocket.CloseMessage)
				_ = ws.WriteMessage(websocket.CloseMessage, []byte{})
				//消息通道关闭后直接注销客户端
				return
			}

			err := ws.WriteMessage(websocket.TextMessage, mm)
			if err != nil {
				fmt.Println("webSocket write error")
				return
			}
		}
	}()

	//for {
	//
	//}

	//开始通信
	//for {
	//	fmt.Println("start transfer message")
	//	msgType, msg, err := ws.ReadMessage()
	//	if err != nil {
	//		fmt.Println("webSocket read error")
	//		return
	//	}
	//	err = ws.WriteMessage(msgType, msg)
	//	if err != nil {
	//		fmt.Println("webSocket write error")
	//		return
	//	}
	//}
}

// ws://127.0.0.1:9090/wsTest
func WebsocketServer() {
	r := gin.Default()

	// 方法一: 直接创建客户端
	r.GET("/wsTest", NewConnection)
	// 方法二: 通过对象创建  客户端连接
	r.GET("/wsTest1", WsClient)
	r.GET("/", func(ctx *gin.Context) {
		ctx.Writer.WriteString("ok")
	})

	clientCount = 0
	r.Run("127.0.0.1:9090")
}

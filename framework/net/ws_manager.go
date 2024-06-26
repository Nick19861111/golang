package net

import (
	"common/logs"
	"encoding/json"
	"errors"
	"fmt"
	"framework/game"
	"framework/protocol"
	"framework/remote"
	"github.com/gorilla/websocket"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

// websocket的相关配置
var (
	websocketUpgrade = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

//end

type CheckOriginHandler func(r *http.Request) bool
type Manager struct {
	sync.RWMutex
	websocketUpgrade   *websocket.Upgrader
	ServerId           string
	CheckOriginHandler CheckOriginHandler
	clients            map[string]Connection //客户端
	ClientReadChan     chan *MsgPack
	handlers           map[protocol.PackageType]EventHandler
	ConnectorHandlers  LogicHandler
	RemoteReadChan     chan []byte
	RemoteCli          remote.Client
}
type HandlerFunc func(session *Session, body []byte) (any, error)
type LogicHandler map[string]HandlerFunc
type EventHandler func(packet *protocol.Packet, c Connection) error

func (m *Manager) Run(addr string) {
	go m.clientReadChanHandler()
	go m.remoteReadChanHandler()
	http.HandleFunc("/", m.serveWS)
	//设置不同的消息处理器
	m.setupEventHandlers()
	logs.Fatal("connector listen serve err:%v", http.ListenAndServe(addr, nil))
}

// ws 的相关
func (m *Manager) serveWS(w http.ResponseWriter, r *http.Request) {
	//websocket 基于http
	if m.websocketUpgrade == nil {
		m.websocketUpgrade = &websocketUpgrade
	}
	wsConn, err := m.websocketUpgrade.Upgrade(w, r, nil)
	if err != nil {
		logs.Error("websocketUpgrade.Upgrade err:%v", err)
		return
	}
	//客户端链接到服务器
	client := NewWsConnection(wsConn, m)
	//加入到一个集合里面
	m.addClient(client)
	client.Run()
}

// 把客户端加入到集合里面
func (m *Manager) addClient(client *WsConnection) {
	//这里不明白具体是做什么
	m.Lock()
	defer m.Unlock()
	//end
	m.clients[client.Cid] = client
}

// 删除客户端
func (m *Manager) removeClient(wc *WsConnection) {
	for cid, c := range m.clients {
		if cid == wc.Cid {
			c.Close()
			//必须要删除对应的uid
			delete(m.clients, cid)
		}
	}
}

// 重点方法 客户端读取相关的数据
func (m *Manager) clientReadChanHandler() {
	for {
		select {
		case body, ok := <-m.ClientReadChan:
			if ok {
				//解析数据
				m.decodeClientPack(body)
			}
		}
	}
}

func (m *Manager) decodeClientPack(body *MsgPack) {
	//解析协议
	//logs.Info("receiver message:%v", string(body.Body))
	packet, err := protocol.Decode(body.Body)
	if err != nil {
		logs.Error("decode message err:%v", err)
		return
	}
	if err := m.routeEvent(packet, body.Cid); err != nil {
		logs.Error("routeEvent err:%v", err)
	}
}

func (m *Manager) Close() {
	for cid, v := range m.clients {
		v.Close()
		delete(m.clients, cid)
	}
}

func (m *Manager) routeEvent(packet *protocol.Packet, cid string) error {
	//根据packet.type来做不同的处理  处理器
	conn, ok := m.clients[cid]
	if ok {
		handler, ok := m.handlers[packet.Type]
		if ok {
			return handler(packet, conn)
		} else {
			return errors.New("no packetType found")
		}
	}
	return errors.New("no client found")
}

func (m *Manager) setupEventHandlers() {
	m.handlers[protocol.Handshake] = m.HandshakeHandler
	m.handlers[protocol.HandshakeAck] = m.HandshakeAckHandler
	m.handlers[protocol.Heartbeat] = m.HeartbeatHandler
	m.handlers[protocol.Data] = m.MessageHandler
	m.handlers[protocol.Kick] = m.KickHandler
}

func (m *Manager) HandshakeHandler(packet *protocol.Packet, c Connection) error {
	res := protocol.HandshakeResponse{
		Code: 200,
		Sys: protocol.Sys{
			Heartbeat: 3,
		},
	}
	data, _ := json.Marshal(res)
	buf, err := protocol.Encode(packet.Type, data)
	if err != nil {
		logs.Error("encode packet err:%v", err)
		return err
	}
	return c.SendMessage(buf)
}

func (m *Manager) HandshakeAckHandler(packet *protocol.Packet, c Connection) error {
	logs.Info("receiver handshake ack message...")
	return nil
}

func (m *Manager) HeartbeatHandler(packet *protocol.Packet, c Connection) error {
	logs.Info("receiver heartbeat message:%v", packet.Type)
	var res []byte
	data, _ := json.Marshal(res)
	buf, err := protocol.Encode(packet.Type, data)
	if err != nil {
		logs.Error("encode packet err:%v", err)
		return err
	}
	return c.SendMessage(buf)
}

func (m *Manager) MessageHandler(packet *protocol.Packet, c Connection) error {
	message := packet.MessageBody()
	logs.Info("receiver message body, type=%v, router=%v, data:%v",
		message.Type, message.Route, string(message.Data))
	//connector.entryHandler.entry
	routeStr := message.Route
	routers := strings.Split(routeStr, ".")
	if len(routers) != 3 {
		return errors.New("router unsupported")
	}
	serverType := routers[0]
	handlerMethod := fmt.Sprintf("%s.%s", routers[1], routers[2])
	connectorConfig := game.Conf.GetConnectorByServerType(serverType)
	if connectorConfig != nil {
		//本地connector服务器处理
		handler, ok := m.ConnectorHandlers[handlerMethod]
		if ok {
			data, err := handler(c.GetSession(), message.Data)
			if err != nil {
				return err
			}
			marshal, _ := json.Marshal(data)
			message.Type = protocol.Response
			message.Data = marshal
			encode, err := protocol.MessageEncode(message)
			if err != nil {
				return err
			}
			res, err := protocol.Encode(packet.Type, encode)
			if err != nil {
				return err
			}
			return c.SendMessage(res)
		}
	} else {
		//nats 远端调用处理 hall.userHandler.updateUserAddress
		dst, err := m.selectDst(serverType)
		if err != nil {
			logs.Error("remote send msg selectDst err:%v", err)
			return err
		}
		msg := &remote.Msg{
			Cid:         c.GetSession().Cid,
			Uid:         c.GetSession().Uid,
			Src:         m.ServerId,
			Dst:         dst,
			Router:      handlerMethod,
			Body:        message,
			SessionData: c.GetSession().data,
		}
		data, _ := json.Marshal(msg)
		err = m.RemoteCli.SendMsg(dst, data)
		if err != nil {
			logs.Error("remote send msg err：%v", err)
			return err
		}
	}
	return nil
}

func (m *Manager) KickHandler(packet *protocol.Packet, c Connection) error {
	logs.Info("receiver kick  message...")
	return nil
}

func (m *Manager) remoteReadChanHandler() {
	for {
		select {
		case msg := <-m.RemoteReadChan:
			logs.Info("sub nats msg:%v", string(msg))
		}
	}
}

func (m *Manager) selectDst(serverType string) (string, error) {
	serversConfigs, ok := game.Conf.ServersConf.TypeServer[serverType]
	if !ok {
		return "", errors.New("no server found")
	}
	//随机一个 比较好的一个策略
	rand.New(rand.NewSource(time.Now().UnixNano()))
	index := rand.Intn(len(serversConfigs))
	return serversConfigs[index].ID, nil
}

func NewManager() *Manager {
	return &Manager{
		ClientReadChan: make(chan *MsgPack, 1024),
		clients:        make(map[string]Connection),
		handlers:       make(map[protocol.PackageType]EventHandler),
		RemoteReadChan: make(chan []byte, 1024),
	}
}

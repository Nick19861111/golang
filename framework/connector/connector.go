package connector

type Connector struct {
	isRuning bool
}

func Default() *Connector {
	return &Connector{}
}

func (c *Connector) Run() {
	if !c.isRuning {
		//启动websocket 和nats相关操作
	}
}

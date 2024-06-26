package game

import (
	"common/logs"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"io"
	"log"
	"os"
	"path"
)

/**
读取配置文件相关操作
*/

import (
	"encoding/json"
)

var Conf *Config

// 对应的文件名
const (
	gameConfig = "gameConfig.json"
	servers    = "servers.json"
)

type Config struct {
	GameConfig  map[string]GameConfigValue `json:"gameConfig"`
	ServersConf ServersConf                `json:"serversConf"`
}
type ServersConf struct {
	Nats       NatsConfig         `json:"nats" `
	Connector  []*ConnectorConfig `json:"connector" `
	Servers    []*ServersConfig   `json:"servers" `
	TypeServer map[string][]*ServersConfig
}

type ServersConfig struct {
	ID               string `json:"id" `
	ServerType       string `json:"serverType" `
	HandleTimeOut    int    `json:"handleTimeOut" `
	RPCTimeOut       int    `json:"rpcTimeOut" `
	MaxRunRoutineNum int    `json:"maxRunRoutineNum" `
}

type ConnectorConfig struct {
	ID         string `json:"id" `
	Host       string `json:"host" `
	ClientPort int    `json:"clientPort" `
	Frontend   bool   `json:"frontend" `
	ServerType string `json:"serverType" `
}
type NatsConfig struct {
	Url string `json:"url" mapstructure:"db"`
}

type GameConfigValue map[string]any

// 初始化读取对应目录的相关文件
func InitConfig(configDir string) {
	Conf = new(Config)
	//通过系统读取目录
	dir, err := os.ReadDir(configDir)
	//读取成功
	if err != nil {
		logs.Fatal("read config dir err:%v", err)
	}
	//循环出对应的json文件进行读取
	for _, v := range dir {
		configFile := path.Join(configDir, v.Name())
		//判断是目录下对应的文件
		if v.Name() == gameConfig {
			readGameConfig(configFile)
		}
		if v.Name() == servers {
			readServersConfig(configFile)
		}
	}
}

func readServersConfig(configFile string) {
	var serversConfig ServersConf
	v := viper.New()
	v.SetConfigFile(configFile)
	v.WatchConfig()
	v.OnConfigChange(func(in fsnotify.Event) {
		log.Println("serversConfig 配置文件被修改了")
		err := v.Unmarshal(&serversConfig)
		if err != nil {
			panic(fmt.Errorf("serversConfig Unmarshal change config data,err:%v \n", err))
		}
		Conf.ServersConf = serversConfig
	})
	err := v.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("serversConfig 读取配置文件出错,err:%v \n", err))
	}
	//解析
	err = v.Unmarshal(&serversConfig)
	if err != nil {
		panic(fmt.Errorf("serversConfig Unmarshal config data,err:%v \n", err))
	}
	Conf.ServersConf = serversConfig
	typeServersConfig()
}

// 读取服务器节点的配置
func typeServersConfig() {
	if len(Conf.ServersConf.Servers) > 0 {
		if Conf.ServersConf.TypeServer == nil {
			Conf.ServersConf.TypeServer = make(map[string][]*ServersConfig)
		}
		for _, v := range Conf.ServersConf.Servers {
			if Conf.ServersConf.TypeServer[v.ServerType] == nil {
				Conf.ServersConf.TypeServer[v.ServerType] = make([]*ServersConfig, 0)
			}
			Conf.ServersConf.TypeServer[v.ServerType] = append(Conf.ServersConf.TypeServer[v.ServerType], v)
		}
	}
}

// 读取游戏的配置
func readGameConfig(configFile string) {
	//gc := make(map[string]GameConfigValue)
	//v := viper.New()
	//v.SetConfigFile(configFile)
	//v.WatchConfig()
	//v.OnConfigChange(func(in fsnotify.Event) {
	//	log.Println("gameConfig配置文件被修改了")
	//	err := v.Unmarshal(&gc)
	//	if err != nil {
	//		panic(fmt.Errorf("gameConfig Unmarshal change config data,err:%v \n", err))
	//	}
	//	Conf.GameConfig = gc
	//})
	//err := v.ReadInConfig()
	//if err != nil {
	//	panic(fmt.Errorf("gameConfig 读取配置文件出错,err:%v \n", err))
	//}
	//log.Println("%v", v.AllKeys())
	//err = v.Unmarshal(&gc)
	//if err != nil {
	//	panic(fmt.Errorf("gameConfig Unmarshal config data,err:%v \n", err))
	//}
	//Conf.GameConfig = gc
	file, err := os.Open(configFile)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}
	var gc map[string]GameConfigValue
	//解析对应的数据
	err = json.Unmarshal(data, &gc)
	if err != nil {
		panic(err)
	}
	Conf.GameConfig = gc
}

// 获得Connector 集群
func (c *Config) GetConnector(serverId string) *ConnectorConfig {
	for _, v := range c.ServersConf.Connector {
		if v.ID == serverId {
			return v
		}
	}
	return nil
}

func (c *Config) GetConnectorByServerType(serverType string) *ConnectorConfig {
	for _, v := range c.ServersConf.Connector {
		if v.ServerType == serverType {
			return v
		}
	}
	return nil
}

func (c *Config) GetFrontGameConfig() map[string]any {
	result := make(map[string]any)
	for k, v := range c.GameConfig {
		value, ok := v["value"]
		backend := false
		_, exist := v["backend"]
		if exist {
			backend = v["backend"].(bool)
		}
		if ok && !backend {
			result[k] = value
		}
	}
	return result
}

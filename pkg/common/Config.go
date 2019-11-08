package common

import (
	"encoding/json"
	"io/ioutil"
)

// 程序配置
type Config struct {
	K8sVersions       []string           `json:"k8sVersions"`
}

var (
	// 单例
	G_config *Config
)

// 加载配置
func InitConfig(file string) (err error) {

	var (
		content []byte
		conf Config
	)
	if file == "" {
		file = "/etc/ansible/config.json"
	}

	// 1, 把配置文件读进来
	if content, err = ioutil.ReadFile(file); err != nil {
		return
	}

	// 2, 做JSON反序列化
	if err = json.Unmarshal(content, &conf); err != nil {
		return
	}

	// 3, 赋值单例
	G_config = &conf

	return
}

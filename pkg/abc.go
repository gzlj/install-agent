package pkg

type Config struct {

	str string `json"str"`
}

var (
	// 单例
	G_config *Config
)

package worker

import "context"

// 程序配置
type Config struct {
	EtcdEndpoints []string `json:"etcdEndpoint"`
	EtcdDialTimeout int `json:"etcdDialTimeout"`
	JobLogBatchSize int `json:"jobLogBatchSize"`
	JobLogCommitTimeout int `json"jobLogCommitTimeout"`
	Cancel context.CancelFunc `json"cancel"`
	DownloadUrl string `json:"downloadUrl"`
	LanRegistry string `json:"lanRegistry"`
	JobId string `json:"jobId"`
}

var (
	// 单例
	G_config *Config
)

// 加载配置
func InitConfig(etcdHost, masterIp, lanRegistry, downloadUrl, jobId string) (err error) {
	/*
	EtcdHost string
	MongoHost string
	MasterIp string
	LanRegistry string
	DownloadUrl string
	 */
	conf := Config{
		EtcdEndpoints: []string{etcdHost},
		EtcdDialTimeout: 5000,
		DownloadUrl: downloadUrl,
		LanRegistry: lanRegistry,
		JobId: jobId,
	}
	// 3, 赋值单例
	G_config = &conf

	return
}
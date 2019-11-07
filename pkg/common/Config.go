package common

//// 程序配置
//type Config struct {
//	EtcdEndpoints       []string           `json:"etcdEndpoint"`
//	EtcdDialTimeout     int                `json:"etcdDialTimeout"`
//}
//
//var (
//	// 单例
//	G_config *Config
//)
//
//// 加载配置
///*func InitConfig(etcdHost string) (err error) {
//	/*
//		EtcdHost string
//		MongoHost string
//		MasterIp string
//		LanRegistry string
//		DownloadUrl string
//	*/
//	conf := Config{
//		EtcdEndpoints:   []string{etcdHost},
//		EtcdDialTimeout: 5000,
//	}
//	// 3, 赋值单例
//	G_config = &conf
//
//	return
//}

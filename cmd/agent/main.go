package main

import (
	"github.com/gin-gonic/gin"
	"github.com/gzlj/install-agent/pkg/agent/handle"
	"github.com/gzlj/install-agent/pkg/common"
	"os"
	"runtime"
)

type APIServer struct {
	engine *gin.Engine
}

// 初始化线程数量
func initEnv() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

/*var (
	EtcdHost string
	MongoHost string
	MasterIp string
	LanRegistry string
	DownloadUrl string
	JobId string
	JobType string	//"node", "sing_master", "ha_master"
	done chan struct{}
	undone chan string
)

func initArgs() {

	flag.StringVar(&EtcdHost, "etcd-host", "127.0.0.1:2379", "etcd db")
	flag.StringVar(&MongoHost, "mongo-host", "./worker.json", "mogon db")
	flag.StringVar(&MasterIp, "master-ip", "127.0.0.1", "k8s master ip")
	flag.StringVar(&LanRegistry, "lan-registry", "127.0.0.1:5000", "lan docker image registry")
	flag.StringVar(&JobId, "job-id", "8888", "job id")
	flag.StringVar(&DownloadUrl, "download-url", "", "download-url")
	flag.StringVar(&JobType, "job-type", "node", "master: install master; node: install node.")
	flag.Parse()


}*/

func (s *APIServer) registryApi() {
	registryBootstrap(s.engine)
}

func registryBootstrap(r *gin.Engine) {
	r.POST("/start", handle.StartTask)
	r.GET("/log",handle.QueryJobLog)
	r.GET("/kubeconfig",handle.QueryKubeconfig)
	r.GET("/status",handle.QueryJobStatus)
	r.GET("/config",handle.ShowConfig)
	r.GET("/liststatus",handle.ListJobStatuses)
	r.POST("/cancel", handle.CancelJob)
	//r.GET("/updatestatus",handle.UpdateJobStatuse)
	//r.GET("/count",handle.GetJobProcess)
}

func init() {
	/*
	//InitConfig
	//etcdHost, masterIp, lanRegistry, downloadUrl, jobId string
	if err = worker.InitConfig(EtcdHost,MasterIp,LanRegistry, DownloadUrl, JobId  ); err != nil {
		//close
		fmt.Println("error InitConfig()")
		os.Exit(-1)
	}
	 */
	//etcdHost := os.Getenv("ETCDHOST")
	//common.InitConfig(etcdHost)
	initEnv()
	configFile := os.Getenv("CONFIGFILE")
	common.InitConfig(configFile)

	/*ip := os.Getenv("ORACLE_SERVER_IP")
	port := os.Getenv("ORACLE_SERVER_PORT")
	//ip string, port string, dbName, user, passwd string
	dbName := os.Getenv("DB_NAME")
	user := os.Getenv("ORACLE_USER")
	passwd := os.Getenv("ORACLE_PASSWORD")

	common.InitOracleDb(ip, port, dbName, user, passwd)*/
	//handle.InitRegister()
}
//
//func initConfig(file string) (err error){
//	if file == "" {
//		file = "/etc/ansible/config.json"
//	}
//
//	return
//}

func main() {


	server := &APIServer{
		engine: gin.Default(),
	}
	server.registryApi()
	server.engine.Run(":18080")
}

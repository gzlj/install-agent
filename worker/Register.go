package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"net"
	"time"
)

// 注册节点到etcd： /cron/workers/IP地址
type Register struct {
	client *clientv3.Client
	kv clientv3.KV
	lease clientv3.Lease

	localIP string // 本机IP
}

var (
	G_register *Register
)

// 获取本机网卡IP
func getLocalIP() (ipv4 string, err error) {
	var (
		addrs []net.Addr
		addr net.Addr
		ipNet *net.IPNet // IP地址
		isIpNet bool

	)
	// 获取所有网卡
	if addrs, err = net.InterfaceAddrs(); err != nil {
		return
	}
	// 取第一个非lo的网卡IP
	for _, addr = range addrs {
		// 这个网络地址是IP地址: ipv4, ipv6
		if ipNet, isIpNet = addr.(*net.IPNet); isIpNet && !ipNet.IP.IsLoopback() {
			// 跳过IPV6
			if ipNet.IP.To4() != nil {
				ipv4 = ipNet.IP.String()	// 192.168.1.1
				return
			}
		}
	}

	err = ERR_NO_LOCAL_IP_FOUND
	return
}

// 注册到/cron/workers/IP, 并自动续租
func (register *Register) keepOnline() {
	fmt.Println("keepOnline start.!!!")
	var (
		regKey string
		leaseGrantResp *clientv3.LeaseGrantResponse
		err error
		keepAliveChan <- chan *clientv3.LeaseKeepAliveResponse
		keepAliveResp *clientv3.LeaseKeepAliveResponse
		cancelCtx context.Context
		cancelFunc context.CancelFunc
	)

	for {
		fmt.Println("keepOnline loop.!!!")
		// 注册路径
		regKey = JOB_WORKER_DIR + register.localIP
		fmt.Println(regKey)
		cancelFunc = nil

		// 创建租约
		if leaseGrantResp, err = register.lease.Grant(context.TODO(), 10); err != nil {
			goto RETRY
		}

		// 自动续租
		if keepAliveChan, err = register.lease.KeepAlive(context.TODO(), leaseGrantResp.ID); err != nil {
			goto RETRY
		}

		cancelCtx, cancelFunc = context.WithCancel(context.TODO())

		// 注册到etcd
		if _, err = register.kv.Put(cancelCtx, regKey, "", clientv3.WithLease(leaseGrantResp.ID)); err != nil {
			fmt.Println("registry fail.!!!")
			goto RETRY
		}


		// 处理续租应答
		for {
			select {
			case keepAliveResp = <- keepAliveChan:
				if keepAliveResp == nil {	// 续租失败
					fmt.Println("keepAliveChan fail.!!!")
					goto RETRY
				}
			}
		}

		RETRY:
		time.Sleep(1 * time.Second)
		if cancelFunc != nil {
			cancelFunc()
		}
	}
}

func InitRegister() (err error) {
	var (
		config clientv3.Config
		client *clientv3.Client
		kv clientv3.KV
		lease clientv3.Lease
		localIp string
	)

	// 初始化配置
	config = clientv3.Config{
		Endpoints: G_config.EtcdEndpoints, // 集群地址
		DialTimeout: time.Duration(G_config.EtcdDialTimeout) * time.Millisecond, // 连接超时
	}

	// 建立连接
	if client, err = clientv3.New(config); err != nil {
		return
	}

	// 本机IP
	if localIp, err = getLocalIP(); err != nil {
		return
	}

	// 得到KV和Lease的API子集
	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)

	G_register = &Register{
		client: client,
		kv: kv,
		lease: lease,
		localIP: localIp,
	}

	// 服务注册
	go G_register.keepOnline()
	go G_register.watchKiller()

	return
}

//listen killing job event
// 监听强杀任务通知
func (register *Register) watchKiller() {
	var (
		watchChan clientv3.WatchChan
		watchResp clientv3.WatchResponse
		watchEvent *clientv3.Event
		jobId string
	)
	// 监听/cron/killer目录
	go func() { // 监听协程
		// 监听/cron/killer/目录的变化
		watcher := clientv3.NewWatcher(register.client)
		watchChan = watcher.Watch(context.TODO(), JOB_KILLER_DIR, clientv3.WithPrefix())
		// 处理监听事件
		for watchResp = range watchChan {
			for _, watchEvent = range watchResp.Events {
				switch watchEvent.Type {
				case mvccpb.PUT:
					jobId = ExtractKillerName(string(watchEvent.Kv.Key))
					//kill the job for jobId
					fmt.Println("jobId: "+ jobId)
					if jobId == G_config.JobId {
						fmt.Println("excute G_config.Cancel()**+++++++")
						G_config.Cancel()
						G_undone <- "cancel k8s-install job."
					}

				case mvccpb.DELETE:
				default:

				}
			}
		}
	}()
}

func UploadStatus(jobId string, status Status) (err error){
	var (
		bytes []byte
	)

	regKey := JOB_SAVE_DIR + jobId
	//json.Marshaler()
	if bytes, err = json.Marshal(status); err != nil {
		fmt.Println("upload job status fail.")
		return
	}

	if _, err = G_register.kv.Put(context.TODO(), regKey, string(bytes)); err != nil {
		fmt.Println("upload job status fail.")
		err = UPLOAD_STATUS_FAIL
	}
	return
}
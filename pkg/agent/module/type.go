package module

/*
[master1]
192.168.25.202

[master1:vars]
lan_registry=192.168.30.67:5000
master_ip=192.168.25.202
kubernetes_version=v1.14.2
pod_subnet=10.244.0.0/16
control_plane_endpoint=192.168.25.199:26443
kube_rpm_version=1.14.2-0
apiserver_lb=192.168.25.199
token=b99a00.a144ef80536d4344
apiserver_lbport=26443
lb_interface=ens33
master3_ip=192.168.25.200
master2_ip=192.168.25.201
master1_ip=192.168.25.202
*/

type InstallConfig struct {
	//Id uint	`json:"id"`
	//Ua string `json:"ua"`
	//Title string `json:"title"`
	//Ip string `json:"ip"`
	//CreatedAt time.Time `json:"createdAt"`
	//UpdatedAt time.Time `json:"updatedAt"`
	JobId                string `json:"jobId"`
	IsHa                 bool   `json:"isHa"`
	LanRegistry          string `json:"lanRegistry"`
	MasterIp             string `json:"masterIp"`
	KubernetesVersion    string `json:"kubernetesVersion"`
	PodSubnet            string `json:"podSubnet"`
	ControlPlaneEndpoint string `json:"controlPlaneEndpoint"`
	KubeRpmVersion       string `json:"kubeRpmVersion"`
	ApiserverLb          string `json:"apiserverLb"`
	Token                string `json:"token"`
	ApiserverLbport      string `json:"apiserverLbport"`
	LbInterface          string `json:"lbInterface"`
	Master3Ip            string `json:"master3Ip"`
	Master2Ip            string `json:"master2Ip"`
	Master1Ip            string `json:"master1Ip"`
	CommonPassword       string `json:"commonPassword"`
	ServiceSubnet       string `json:"serviceSubnet"`
}

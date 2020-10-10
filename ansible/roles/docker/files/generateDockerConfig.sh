#/bin/bash

OPTS=$(getopt -o : --long insecure-registries:,docker-home: -- "$@")
if [ $? != 0 ]; then
  echo -e "\033[31m[ERROR] centos_master.sh argument is illegal\033[0m"
fi
eval set -- "$OPTS"
while true ; do
  case "$1" in
    --insecure-registries) insecureRegistries=$2; shift 2;;
    --docker-home) dockerHome=$2; shift 2;;
    --) shift; break;;
  esac
done

if [ -z "$dockerHome" ]; then
  dockerHome=/var/lib/docker
fi

array=(`echo $insecureRegistries | tr ',' ' '`)
arrayStr=
for var in ${array[@]}
do 
arrayStr=$arrayStr\"$var\",
done
arrayStr=$( echo $arrayStr | sed 's#.$##g' )

mkdir -p /etc/docker
cat > /etc/docker/daemon.json << EOF
{
      "registry-mirrors": ["https://bxsfpjcb.mirror.aliyuncs.com", "https://registry.docker-cn.com"],
      "max-concurrent-downloads": 10,
      "log-driver": "json-file",
      "log-level": "warn",
      "log-opts": {
          "max-size": "100m",
          "max-file": "3"
      },
      "exec-opts": ["native.cgroupdriver=systemd"],
      "insecure-registries": [ $arrayStr ],
      "graph": "$dockerHome"
}
EOF

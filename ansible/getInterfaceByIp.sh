#/bin/bash

# all interfaces
target_ip=$1
interfaces=($(ip addr show | grep ^[0-9]*: | awk '{print $2}' | awk -F ":" '{print $1}' | egrep -v "(^veth|^docker|^flannel)"))
#echo "interfaces= ${interfaces[@]}"
#interfaces=(ens33 docker0)

for ((i = 0; i < ${#interfaces[*]}; i++))
do
  interface=${interfaces[i]}
  ip=($(ip addr show $interface | grep "inet\b" | awk '{print $2}' | awk -F "/" '{print $1}'))
  for(( j = 0; j < ${#ip[*]}; j++ ))
  do
    if [ "${ip[j]}" == "$target_ip" ]; then
      result_interface=$interface
    fi
  done
done
echo $result_interface

#/bin/bash

array=($*)
[ -f  ~/.ssh/id_rsa ] || ssh-keygen -P "" -f ~/.ssh/id_rsa -t rsa -b 1024

password=${array[0]}
max_inde=$(( ${#array[*]} -1 ))
for (( i = 1; i < ${#array[*]}; i++ ))
do
/usr/bin/sshpass -p $password ssh-copy-id -o StrictHostKeyChecking=no root@${array[i]}

done

#do
#  echo ${array[i]}
#  /usr/bin/expect <<-EOF
#  spawn ssh-copy-id root@${array[i]}
#  expect {
#  "yes/no" { send "yes\r"; exp_continue }
#  "password" { send "$password\r"; }
#  }
#EOF
#done
#/usr/bin/expect <<-EOF
#spawn ssh-copy-id root@${array[$max_inde]}
#expect {
#"yes/no" { send "yes\r"; exp_continue }
#"password:" { send "$password\r"; }
#}
#expect eof
#EOF
exit 0


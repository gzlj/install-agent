#/bin/bash

array=($*)
[ -f  ~/.ssh/id_rsa ] || ssh-keygen -P "" -f ~/.ssh/id_rsa -t rsa -b 1024

for (( i = 1; i < ${#array[*]}; i++ ))
do
  echo ${array[i]}
  /usr/bin/expect <<EOF
  spawn ssh-copy-id root@${array[i]}
  expect "yes/no"
  send "yes\r"
  expect "password"
  send "${array[0]}\r"
EOF
done

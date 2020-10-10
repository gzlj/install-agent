FROM centos:7.6.1810
ADD agent /
COPY ansible /etc/ansible
RUN yum install yum-utils -y && yum-config-manager --add-repo http://mirrors.aliyun.com/repo/epel-7.repo
RUN yum install ansible -y
RUN yum install openssh-clients sshpass expect -y
#RUN rm -fr /etc/ansible && /bin/mv -f /ansible /etc/
CMD ["/agent"]

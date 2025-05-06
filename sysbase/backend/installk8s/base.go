package installk8s

import (
	"fmt"
	"net"
	"strings"
	"time"

	"sysbase/model"
	"sysbase/tool/execremote"
)

type (
	InstallK8s struct {
		SourceDir string
		Params    Params
		Stdout    chan string
		Defer     func()
		AddNew    bool
		OnlyConf  bool
		UseLvs    bool
		er        *execremote.ExecRemote
		resources map[string]execremote.Role
	}
	Params struct {
		K8sClusterID uint
		DoWhat       string
		DoContainerd bool
	}
)

var localIps []string

func (ik *InstallK8s) Call(funcName string) {
	defer func() {
		ik.Defer()
		ik.er.Close()
	}()

	ik.Stdout <- "开始执行..."

	switch funcName {
	case "InstallTest":
		ik.InstallTest()
		break
	case "InstallAll":
		ik.InstallAll()
		break
	case "InstallBase":
		ik.InstallBase()
		break
	case "UpdateKernel":
		ik.UpdateKernel()
		break
	case "InstallBaseBin":
		ik.InstallBaseBin()
		break
	case "InstallDns":
		ik.InstallDns()
		break
	case "InstallContainerd":
		ik.InstallContainerd()
		break
	case "InstallRegistry":
		ik.InstallRegistry()
		break
	case "InstallEtcd":
		ik.InstallEtcd()
		break
	case "InstallMaster":
		ik.InstallMaster()
		break
	case "InstallNode":
		ik.InstallNode()
		break
	case "InstallContainerdCrt":
		ik.InstallContainerdCrt()
		break
	case "InstallLvs":
		ik.InstallLvs()
		break
	case "ServicePublish":
		ik.ServicePublish()
		break
	case "ServiceEtcd":
		ik.ServiceEtcd()
		break
	case "ServiceMaster":
		ik.ServiceMaster()
		break
	case "ServiceNode":
		ik.ServiceNode()
		break
	case "ServiceDns":
		ik.ServiceDns()
		break
	case "FinishInstall":
		ik.FinishInstall()
		break
	case "NewnodeInstall":
		ik.NewnodeInstall()
		break
	case "NewetcdInstall":
		ik.NewetcdInstall()
		break
	case "NewmasterInstall":
		ik.NewmasterInstall()
		break
	case "UpdateSslMaster":
		ik.UpdateSslMaster()
		break
	case "UpdateSslEtcd":
		ik.UpdateSslEtcd()
		break
	case "UpdateSslNode":
		ik.UpdateSslNode()
		break
	default:
		ik.Stdout <- fmt.Sprintf("没有%s的Call方法", funcName)
	}

	ik.Stdout <- "结束执行"
	time.Sleep(1 * time.Millisecond)
}

func (ik *InstallK8s) GetResources() {
	localIps, _ = getLocalIp()

	k8sClusterResource := model.K8sClusterResource{}

	resource, err := k8sClusterResource.ListResource(ik.Params.K8sClusterID, []string{})
	if err != nil {
		ik.Stdout <- err.Error()
		return
	}

	var user, password string
	ik.resources = make(map[string]execremote.Role)
	for _, r := range resource {
		user = r.User
		password = r.Password
		role := ik.resources[r.Scope]
		role.Name = r.Scope
		role.Parallel = true
		role.Hosts = append(role.Hosts, fmt.Sprintf("%s:%d", r.Host, r.Port))
		ik.resources[r.Scope] = role
	}

	timeout := 3 * time.Second
	ik.er = execremote.New(user, password, timeout, ik.Stdout)
}

func (ik *InstallK8s) InstallTest() {
	publishRole, ok := ik.resources["publish"]
	if !ok {
		ik.Stdout <- "没有publish资源"
		return
	}

	ik.er.SetRole(publishRole)

	cmds := []string{
		`apt-get install git -y`,
		`apt-get remove git -y`,
	}

	ik.er.Run(cmds...)
}

func (ik *InstallK8s) InstallAll() {
	var hosts []string
	if r, ok := ik.resources["publish"]; ok {
		hosts = append(hosts, r.Hosts...)
	}
	if r, ok := ik.resources["etcd"]; ok {
		hosts = append(hosts, r.Hosts...)
	}
	if r, ok := ik.resources["master"]; ok {
		hosts = append(hosts, r.Hosts...)
	}
	if r, ok := ik.resources["node"]; ok {
		hosts = append(hosts, r.Hosts...)
	}
	if r, ok := ik.resources["registry"]; ok {
		hosts = append(hosts, r.Hosts...)
	}
	if r, ok := ik.resources["lvs"]; ok {
		hosts = append(hosts, r.Hosts...)
	}

	needUpdateHosts := ik.checkHostsKernel(hosts)
	if len(needUpdateHosts) > 0 {
		ik.Stdout <- fmt.Sprintf("请先确保这些机器内核已升级:%v\n", needUpdateHosts)
		//ik.UpdateKernel()
		return
	}
	ik.InstallBase()
	ik.InstallBaseBin()
	ik.InstallDns()
	ik.InstallContainerd()
	ik.InstallRegistry()
	ik.InstallEtcd()
	ik.InstallMaster()
	ik.InstallNode()
	ik.InstallContainerdCrt()
	ik.InstallLvs()
	ik.FinishInstall()
}

func (ik *InstallK8s) InstallBase() {
	var role []execremote.Role

	if r, ok := ik.resources["publish"]; ok {
		role = append(role, r)
	}
	if r, ok := ik.resources["etcd"]; ok {
		role = append(role, r)
	}
	if r, ok := ik.resources["master"]; ok {
		role = append(role, r)
	}
	if r, ok := ik.resources["node"]; ok {
		role = append(role, r)
	}
	if r, ok := ik.resources["registry"]; ok {
		role = append(role, r)
	}
	if r, ok := ik.resources["lvs"]; ok {
		role = append(role, r)
	}

	ik.er.SetRole(role...)
	ik.installBase()
}

func (ik *InstallK8s) installBase() {
	//cmds := []string{
	//	fmt.Sprintf(`chown -R root:root %s`, ik.SourceDir),
	//	"apt-get install -y telnet net-tools openssl socat",
	//	"mkdir -p /data/apps > /dev/null 2>&1;if [ $? == 0 ];then useradd -d /data/apps/www esn && useradd -d /data/apps/www www && usermod -G esn www && chmod 750 /data/apps/www && mkdir -p /data/apps/log/nginx && chown -R www:www /data/apps/log && chmod 750 /data/apps/log;fi",
	//	"systemctl stop firewalld && systemctl disable firewalld",
	//	`sed -i "s#SELINUX=enforcing#SELINUX=disabled#g" /etc/selinux/config && setenforce 0`,
	//	`cat /etc/sysctl.conf | grep net.ipv4.ip_forward > /dev/null 2>&1 ; if [ $? -ne 0 ];then echo "net.ipv4.ip_forward = 1" >> /etc/sysctl.conf && sysctl -p;fi`,
	//	`cat /etc/sysctl.conf | grep net.ipv4.conf.all.rp_filter > /dev/null 2>&1 ; if [ $? -ne 0 ];then echo "net.ipv4.conf.all.rp_filter = 1" >> /etc/sysctl.conf && sysctl -p;fi`,
	//}

	cmds := []string{
		fmt.Sprintf(`chown -R root:root %s`, ik.SourceDir),
		"apt-get install -y telnet net-tools openssl socat",
		"mkdir -p /data/apps > /dev/null 2>&1; if [ $? == 0 ]; then useradd -d /data/apps/www esn && useradd -d /data/apps/www www && usermod -G esn www && chmod 750 /data/apps/www && mkdir -p /data/apps/log/nginx && chown -R www:www /data/apps/log && chmod 750 /data/apps/log; fi",
		//"systemctl stop ufw && systemctl disable ufw",  // Ubuntu 使用 ufw 而不是 firewalld
		//`sed -i "s#SELINUX=enforcing#SELINUX=disabled#g" /etc/selinux/config 2>/dev/null || true`, // Ubuntu 默认没有 SELinux
		`grep -q "net.ipv4.ip_forward" /etc/sysctl.conf || echo "net.ipv4.ip_forward = 1" >> /etc/sysctl.conf && sysctl -p`,
		`grep -q "net.ipv4.conf.all.rp_filter" /etc/sysctl.conf || echo "net.ipv4.conf.all.rp_filter = 1" >> /etc/sysctl.conf && sysctl -p`,
	}
	ik.er.Run(cmds...)
}

func (ik *InstallK8s) UpdateKernel() {
	var role []execremote.Role

	if r, ok := ik.resources["etcd"]; ok {
		role = append(role, r)
	}
	if r, ok := ik.resources["master"]; ok {
		role = append(role, r)
	}
	if r, ok := ik.resources["node"]; ok {
		role = append(role, r)
	}
	if r, ok := ik.resources["registry"]; ok {
		role = append(role, r)
	}
	if r, ok := ik.resources["lvs"]; ok {
		role = append(role, r)
	}

	ik.er.SetRole(role...)
	ik.updateKernel()
}

func (ik *InstallK8s) checkHostsKernel(hosts []string) []string {
	// 检查每个节点的内核，如果不是最新内核，则升级内核
	var needUpdateHosts []string
	for _, host := range hosts {
		oneHostRole := execremote.Role{
			Hosts:      []string{host},
			WaitOutput: true,
		}
		ik.er.SetRole(oneHostRole)
		// 当前内核版本小于 4.19.94，则升级内核
		ik.er.Run(`[ "$(printf '%s\n' "4.19.94" "$(uname -r | cut -d'-' -f1)" | sort -V | head -n1)" != "4.19.94" ] && echo 1`)
		rets := ik.er.GetCmdReturn()
		if len(rets) > 0 && ik.er.GetCmdReturn()[0] == "1" {
			oneHostRole.WaitOutput = false
			needUpdateHosts = append(needUpdateHosts, host)
		}
	}

	return needUpdateHosts
}

func (ik *InstallK8s) updateKernel() {
	//ik.er.Put(fmt.Sprintf("%s/kernel-4.19.94.gz", ik.SourceDir), "/tmp/kernel-4.19.94.gz")
	//ik.er.Run("cd /tmp && tar zxvf kernel-4.19.94.gz && yum remove -y kernel-headers kernel-tools-libs kernel-tools kernel-ml-tools kernel-ml-tools-libs && yum install -y kernel-4.19.94/* ; rm -rf kernel-4.19.94*")
	//ik.er.Run(`num=$(awk -F \' '$1=="menuentry " {print i++ " : " $2}' /etc/grub2.cfg | grep 4.19.94 | awk '{print $1}') && grub2-set-default $num && grub2-mkconfig -o /boot/grub2/grub.cfg ; grub2-editenv list`)
	//ik.er.Run("reboot")

	// 上传内核包
	ik.er.Put(fmt.Sprintf("%s/kernel-4.19.94.gz", ik.SourceDir), "/tmp/kernel-4.19.94.gz")

	// 判断系统类型并执行对应命令
	ik.er.Run(`
    if [ -f /etc/redhat-release ]; then
        # CentOS/RHEL
        cd /tmp && tar zxvf kernel-4.19.94.gz &&
        yum remove -y kernel-headers kernel-tools-libs kernel-tools kernel-ml-tools kernel-ml-tools-libs &&
        yum install -y kernel-4.19.94/* &&
        rm -rf kernel-4.19.94* &&
        num=$(awk -F\' '$1=="menuentry " {print i++ " : " $2}' /etc/grub2.cfg | grep 4.19.94 | awk '{print $1}') &&
        grub2-set-default $num &&
        grub2-mkconfig -o /boot/grub2/grub.cfg &&
        grub2-editenv list
    else
        # Ubuntu/Debian
        cd /tmp && tar zxvf kernel-4.19.94.gz &&
        apt remove -y linux-headers-* linux-tools-* &&
        dpkg -i kernel-4.19.94/*.deb &&
        rm -rf kernel-4.19.94* &&
        update-grub &&
        grub-set-default "$(grep -oP "menuentry '\K.*4.19.94.*(?=')" /boot/grub/grub.cfg | head -n 1)" &&
        update-grub
    fi &&
    reboot
`)

}

func (ik *InstallK8s) InstallBaseBin() {
	pridnsRole, ok := ik.resources["pridns"]
	if !ok {
		ik.Stdout <- "没有pridns资源"
		return
	}
	pridnsHost := strings.Split(pridnsRole.Hosts[0], ":")[0]

	var role []execremote.Role

	if r, ok := ik.resources["publish"]; ok {
		role = append(role, r)
	}

	ik.er.SetRole(role...)

	ik.er.Put(fmt.Sprintf("%s/basebin.gz", ik.SourceDir), "/tmp")
	ik.er.Run("tar zxvf /tmp/basebin.gz -C / && rm -rf /tmp/basebin.gz")

	if r, ok := ik.resources["etcdlb"]; ok {
		host := strings.Split(r.Hosts[0], ":")
		cmds := []string{
			"mkdir -p /etc/calico",
			fmt.Sprintf(`echo 'apiVersion: projectcalico.org/v3
kind: CalicoAPIConfig
metadata:
spec:
  etcdEndpoints: "https://%s:2379"
  etcdKeyFile: "/etc/cni/net.d/calico-tls/etcd-key"
  etcdCertFile: "/etc/cni/net.d/calico-tls/etcd-cert"
  etcdCACertFile: "/etc/cni/net.d/calico-tls/etcd-ca"' > /etc/calico/calicoctl.cfg`, host[0]),
		}
		cmds = getModifyDnsCmds(cmds, pridnsHost)
		ik.er.Run(cmds...)
	}
}

func (ik *InstallK8s) InstallLvs() {
	if !ik.UseLvs {
		return
	}

	etcdRole, ok := ik.resources["etcd"]
	if !ok {
		ik.Stdout <- "没有etcd资源"
		return
	}

	masterRole, ok := ik.resources["master"]
	if !ok {
		ik.Stdout <- "没有master资源"
		return
	}

	ik.installLvs()
	ik.installLvsvipEtcd(etcdRole)
	ik.installLvsvipMaster(masterRole)
}

func (ik *InstallK8s) installLvs() {
	if !ik.UseLvs {
		return
	}

	lvsRole, ok := ik.resources["lvs"]
	if !ok {
		ik.Stdout <- "没有lvs资源"
		return
	}

	etcdLbRole, ok := ik.resources["etcdlb"]
	if !ok {
		ik.Stdout <- "没有etcdlb资源"
		return
	}

	masterLbRole, ok := ik.resources["masterlb"]
	if !ok {
		ik.Stdout <- "没有masterlb资源"
		return
	}

	etcdRole, ok := ik.resources["etcd"]
	if !ok {
		ik.Stdout <- "没有etcd资源"
		return
	}

	masterRole, ok := ik.resources["master"]
	if !ok {
		ik.Stdout <- "没有master资源"
		return
	}

	ik.er.SetRole(lvsRole)

	etcdLbHost := strings.Split(etcdLbRole.Hosts[0], ":")[0]
	masterLbHost := strings.Split(masterLbRole.Hosts[0], ":")[0]

	//cmds := []string{
	//	`apt-get install -y ipvsadm`,
	//	`systemctl enable ipvsadm`,
	//	`systemctl start ipvsadm`,
	//	fmt.Sprintf(`ifconfig eth0:lvs:0 %s broadcast %s netmask 255.255.255.255 up && echo -e "#/bin/sh\n# chkconfig:   2345 90 10\nifconfig eth0:lvs:0 %s broadcast %s netmask 255.255.255.255 up" > /etc/rc.d/init.d/vip_route_lvs.sh`, etcdLbHost, etcdLbHost, etcdLbHost, etcdLbHost),
	//	fmt.Sprintf(`ifconfig eth0:lvs:1 %s broadcast %s netmask 255.255.255.255 up && echo "ifconfig eth0:lvs:1 %s broadcast %s netmask 255.255.255.255 up" >> /etc/rc.d/init.d/vip_route_lvs.sh`, masterLbHost, masterLbHost, masterLbHost, masterLbHost),
	//	fmt.Sprintf(`route add -host %s dev eth0:lvs:0 ; echo "" > /dev/null && echo "route add -host %s dev eth0:lvs:0 ; echo "" > /dev/null" >> /etc/rc.d/init.d/vip_route_lvs.sh`, etcdLbHost, etcdLbHost),
	//	fmt.Sprintf(`route add -host %s dev eth0:lvs:1 ; echo "" > /dev/null && echo "route add -host %s dev eth0:lvs:1 ; echo "" > /dev/null" >> /etc/rc.d/init.d/vip_route_lvs.sh`, masterLbHost, masterLbHost),
	//	`chmod +x /etc/rc.d/init.d/vip_route_lvs.sh && chkconfig --add vip_route_lvs.sh && chkconfig vip_route_lvs.sh on`,
	//	`echo "1" > /proc/sys/net/ipv4/ip_forward`,
	//}

	cmds := []string{
		// 安装 ipvsadm
		`apt-get install -y ipvsadm`,
		`systemctl enable ipvsadm`,
		`systemctl start ipvsadm`,

		// TODO 自动获取物理接口名称
		// ip link show | awk -F': ' '/state UP/ {print $2; exit}'
		// ip -o link show | awk -F': ' '{print $2}' | grep -Ev '^(lo|docker|virbr|veth|br-|tun|tap|cali|flannel|cni)'
		// ls /sys/class/net | grep -Ev '^(lo|docker|virbr|veth|br-|tun|tap|cali|flannel|cni)'

		// 创建虚拟接口并持久化配置
		fmt.Sprintf(`ip addr add %s/32 dev eth0 label eth0:lvs0 && `+
			`echo -e "#!/bin/sh\nip addr add %s/32 dev eth0 label eth0:lvs0" > /etc/network/if-up.d/vip_route_lvs`,
			etcdLbHost, etcdLbHost),

		fmt.Sprintf(`ip addr add %s/32 dev eth0 label eth0:lvs1 && `+
			`echo "ip addr add %s/32 dev eth0 label eth0:lvs1" >> /etc/network/if-up.d/vip_route_lvs`,
			masterLbHost, masterLbHost),

		// 添加路由并持久化
		fmt.Sprintf(`ip route add %s/32 dev eth0:lvs0 && `+
			`echo "ip route add %s/32 dev eth0:lvs0" >> /etc/network/if-up.d/vip_route_lvs`,
			etcdLbHost, etcdLbHost),

		fmt.Sprintf(`ip route add %s/32 dev eth0:lvs1 && `+
			`echo "ip route add %s/32 dev eth0:lvs1" >> /etc/network/if-up.d/vip_route_lvs`,
			masterLbHost, masterLbHost),

		// 设置脚本权限
		`chmod +x /etc/network/if-up.d/vip_route_lvs`,

		// 启用IP转发
		`echo "net.ipv4.ip_forward=1" >> /etc/sysctl.conf`,
		`sysctl -p`,
	}

	ik.er.Run(cmds...)

	// etcd
	ipvsadm := fmt.Sprintf(`-A -t %s:2379 -s wrr\n`, etcdLbHost)
	for _, host := range etcdRole.Hosts {
		curHost := strings.Split(host, ":")[0]
		ipvsadm = fmt.Sprintf(`%s-a -t %s:2379 -r %s:2379 -g -w 1\n`, ipvsadm, etcdLbHost, curHost)
	}

	// master
	ipvsadm = fmt.Sprintf(`%s-A -t %s:6443 -s wrr\n`, ipvsadm, masterLbHost)
	for _, host := range masterRole.Hosts {
		curHost := strings.Split(host, ":")[0]
		ipvsadm = fmt.Sprintf(`%s-a -t %s:6443 -r %s:6443 -g -w 1\n`, ipvsadm, masterLbHost, curHost)
	}

	//cmds = []string{
	//	fmt.Sprintf(`echo "%s" > /etc/sysconfig/ipvsadm`, ipvsadm),
	//	`systemctl restart ipvsadm && ipvsadm -Ln`,
	//}

	cmds = []string{
		// 保存规则到 Ubuntu 的默认路径（/etc/ipvsadm.rules）
		fmt.Sprintf(`echo "%s" > /etc/ipvsadm.rules`, ipvsadm),

		// 检查 systemd 是否管理 ipvsadm（Ubuntu 默认不提供 ipvsadm.service，需手动加载）
		`if systemctl list-unit-files | grep -q ipvsadm; then systemctl restart ipvsadm; else ipvsadm-restore < /etc/ipvsadm.rules; fi`,

		// 验证规则是否生效
		`ipvsadm -Ln`,
	}

	ik.er.Run(cmds...)
}

func (ik *InstallK8s) installLvsNew() {
	if !ik.UseLvs {
		return
	}

	etcdLbRole, ok := ik.resources["etcdlb"]
	if !ok {
		ik.Stdout <- "没有etcdlb资源"
		return
	}

	masterLbRole, ok := ik.resources["masterlb"]
	if !ok {
		ik.Stdout <- "没有masterlb资源"
		return
	}

	lvsRole, ok := ik.resources["lvs"]
	if !ok {
		ik.Stdout <- "没有lvs资源"
		return
	}

	etcdRole, ok := ik.resources["etcd"]
	if !ok {
		ik.Stdout <- "没有etcd资源"
		return
	}

	masterRole, ok := ik.resources["master"]
	if !ok {
		ik.Stdout <- "没有master资源"
		return
	}

	etcdLbHost := strings.Split(etcdLbRole.Hosts[0], ":")[0]
	masterLbHost := strings.Split(masterLbRole.Hosts[0], ":")[0]

	ik.er.SetRole(lvsRole)
	ik.er.Run("systemctl stop ipvsadm")

	// etcd
	ipvsadm := fmt.Sprintf(`-A -t %s:2379 -s wrr\n`, etcdLbHost)
	var etcdHosts []string
	etcdHosts = append(etcdHosts, etcdRole.Hosts...)
	if newetcdRole, ok := ik.resources["newetcd"]; ok {
		etcdHosts = append(etcdHosts, newetcdRole.Hosts...)
	}
	for _, host := range etcdHosts {
		curHost := strings.Split(host, ":")[0]
		ipvsadm = fmt.Sprintf(`%s-a -t %s:2379 -r %s:2379 -g -w 1\n`, ipvsadm, etcdLbHost, curHost)
	}

	// master
	ipvsadm = fmt.Sprintf(`-A -t %s:6443 -s wrr\n'`, masterLbHost)
	var masterHosts []string
	masterHosts = append(masterHosts, masterRole.Hosts...)
	if newmasterRole, ok := ik.resources["newmaster"]; ok {
		masterHosts = append(masterHosts, newmasterRole.Hosts...)
	}
	for _, host := range masterHosts {
		curHost := strings.Split(host, ":")[0]
		ipvsadm = fmt.Sprintf(`%s-a -t %s:6443 -r %s:6443 -g -w 1\n`, ipvsadm, masterLbHost, curHost)
	}

	//cmds := []string{
	//	fmt.Sprintf(`echo "%s" > /etc/sysconfig/ipvsadm`, ipvsadm),
	//	`systemctl start ipvsadm && ipvsadm -Ln`,
	//}

	cmds := []string{
		// 保存规则到 Ubuntu 的默认路径（/etc/ipvsadm.rules）
		fmt.Sprintf(`echo "%s" > /etc/ipvsadm.rules`, ipvsadm),

		// 检查 systemd 是否管理 ipvsadm（Ubuntu 默认不提供 ipvsadm.service，需手动加载）
		`if systemctl list-unit-files | grep -q ipvsadm; then systemctl restart ipvsadm; else ipvsadm-restore < /etc/ipvsadm.rules; fi`,

		// 验证规则是否生效
		`ipvsadm -Ln`,
	}

	ik.er.Run(cmds...)
}

func (ik *InstallK8s) InstallDns() {
	pridnsRole, ok := ik.resources["pridns"]
	if !ok {
		ik.Stdout <- "没有pridns资源"
		return
	}

	publishRole, ok := ik.resources["publish"]
	if !ok {
		ik.Stdout <- "没有publish资源"
		return
	}

	registryRole, ok := ik.resources["registry"]
	if !ok {
		ik.Stdout <- "没有registry资源"
		return
	}

	registryHost := strings.Split(registryRole.Hosts[0], ":")[0]

	ik.er.SetRole(pridnsRole)
	//ik.er.Run("yum install -y bind-chroot")
	ik.er.Run("apt-get install -y bind9 bind9utils bind9-doc dnsutils")

	ik.er.SetRole(publishRole)

	cmds := []string{
		fmt.Sprintf(`cd %s/bind && sed -i "s#HOST#%s#g" var/named/zones/registry.k8s.io.zone`, ik.SourceDir, registryHost),
		fmt.Sprintf(`cd %s/bind && tar zcvf bind.gz var etc`, ik.SourceDir),
	}
	ik.er.Run(cmds...)

	// 如果是在发布机上运行，此步骤不需要执行
	if !strInArr(strings.Split(publishRole.Hosts[0], ":")[0], localIps) {
		ik.er.Get(fmt.Sprintf("%s/bind/bind.gz", ik.SourceDir), fmt.Sprintf("%s/bind", ik.SourceDir))
	}

	ik.er.SetRole(pridnsRole)
	ik.er.Put(fmt.Sprintf("%s/bind/bind.gz", ik.SourceDir), "/tmp")
	ik.er.Run("tar zxvf /tmp/bind.gz -C / && rm -rf /tmp/bind.gz && chown -R named:named /var/named/zones && chown root:named /var/named /etc/named.conf /etc/named.rfc1912.zones")
	// systemctl enable --now named-chroot && systemctl restart named-chroot
	ik.er.Run("systemctl start named && systemctl enable named")
	ik.er.Local(fmt.Sprintf("rm -rf %s/bind/bind.gz", ik.SourceDir))

	// 发布机添加DNS记录
	ik.er.SetRole(publishRole)
	ik.er.Run(fmt.Sprintf(`nmcli con mod "$(nmcli -t -f NAME connection show | head -n 1)" ipv4.dns "%s"`, registryHost))

	// systemctl restart NetworkManager
	//ik.er.Run("rndc reload && systemd-resolve --flush-caches")
	ik.er.Run("ufw allow 53 && systemctl restart systemd-networkd")

}

func strInArr(str string, arr []string) bool {
	for _, s := range arr {
		if s == str {
			return true
		}
	}
	return false
}

func getLocalIp() ([]string, error) {
	var ips = []string{"localhost", "127.0.0.1"}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ips, err
	}

	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		// = GET LOCAL IP ADDRESS
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP.String())
			}
		}
	}

	return ips, nil
}

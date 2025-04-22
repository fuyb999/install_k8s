package installk8s

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func getModifyDnsCmds(cmds []string, pridnsHost string) []string {
	// 匹配添加私有DNS配置，确保在匹配行的上面只添加一行，并且不重复添加
	//cmd1 := fmt.Sprintf(`grep "%s" /etc/resolv.conf > /dev/null 2>&1 || (cp -rp /etc/resolv.conf /tmp/resolv.conf && awk 'BEGIN{d=0};{ if($0 ~ /nameserver / && d==0) {d=1; printf("nameserver %s\n%%s\n", $0)} else {print $0}}' /tmp/resolv.conf > /etc/resolv.conf && rm -rf /tmp/resolv.conf)`, pridnsHost, pridnsHost)

	cmd := "apt-get install -y openresolv"

	// 添加私有DNS到 resolv.conf（临时）
	cmd1 := fmt.Sprintf(
		`echo "nameserver %s" | sudo tee -a /etc/resolvconf/resolv.conf.d/head > /dev/null && `+
			`resolvconf -u`,
		pridnsHost,
	)

	// 修改网卡配置文件，添加DNS，避免重启后DNS丢失
	//cmd2 := fmt.Sprintf(`file=$(grep -rl ^DNS $(ls /etc/sysconfig/network-scripts/ifcfg-*)) && (grep "DNS1=%s" $file > /dev/null 2>&1 || (DNS=$(echo DNS1=%s && cat $file | grep ^DNS | awk -F '=' '{print "DNS"(NR+1)"="$2}' && sed -i '/^DNS.*/d' $file); echo "$DNS" >> $file))`, pridnsHost, pridnsHost)

	// 通过 netplan 永久配置DNS
	cmd2 := fmt.Sprintf(
		`sed -i '/nameservers:/!b;n;/%s/!a \ \ \ \ \ \ - %s' `+
			`$(ls /etc/netplan/*.yaml) && `+
			`netplan apply`,
		pridnsHost, pridnsHost,
	)

	return append(cmds, cmd, cmd1, cmd2)
}

func GetLinuxDistribution() string {
	// 1. 检查 /etc/os-release
	if distro := getDistroFromOSRelease(); distro != "unknown" {
		return distro
	}

	// 2. 检查 /etc/issue
	if distro := getDistroFromIssue(); distro != "unknown" {
		return distro
	}

	// 3. 检查包管理器
	if _, err := os.Stat("/usr/bin/apt"); err == nil {
		return "ubuntu"
	} else if _, err := os.Stat("/usr/bin/yum"); err == nil {
		return "centos"
	} else if _, err := os.Stat("/usr/bin/dnf"); err == nil {
		return "centos"
	}

	return "unknown"
}

func getDistroFromOSRelease() string {
	file, err := os.Open("/etc/os-release")
	if err != nil {
		return "unknown"
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ID=") {
			id := strings.TrimPrefix(line, "ID=")
			id = strings.Trim(id, `"`)
			switch id {
			case "ubuntu", "debian":
				return "ubuntu"
			case "centos", "rhel", "fedora":
				return "centos"
			}
		} else if strings.HasPrefix(line, "ID_LIKE=") {
			idLike := strings.TrimPrefix(line, "ID_LIKE=")
			idLike = strings.Trim(idLike, `"`)
			if strings.Contains(idLike, "debian") {
				return "ubuntu"
			} else if strings.Contains(idLike, "rhel") || strings.Contains(idLike, "fedora") {
				return "centos"
			}
		}
	}
	return "unknown"
}

func getDistroFromIssue() string {
	file, err := os.Open("/etc/issue")
	if err != nil {
		return "unknown"
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		line := strings.ToLower(scanner.Text())
		if strings.Contains(line, "ubuntu") {
			return "ubuntu"
		} else if strings.Contains(line, "centos") || strings.Contains(line, "red hat") {
			return "centos"
		}
	}
	return "unknown"
}

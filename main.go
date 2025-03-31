package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	// "sort"
	"strings"
	// "time"
	"gopkg.in/yaml.v3"
	// "strconv"
)

type ProxyConfig struct {
	Name     string
	Type     string
	Server   string
	Port     string
	UUID     string
	AlterId  string
	Cipher   string
	Password string
	SNI      string // 添加 SNI 字段
}

type ProxyGroupConfig struct {
	Name           string
	Type           string
	ProxyNames     []string
	SelectedProxy  string
}

const (
	VERSION = "0.1.0"
)

func main() {
	// 创建命令行子命令
	flag.Usage = func() {
		fmt.Printf("用法: %s <命令> [参数]\n\n", os.Args[0])
		fmt.Println("可用命令:")
		fmt.Println("  install    安装并配置 Clash Premium")
		fmt.Println("  start      启动 Clash 服务")
		fmt.Println("  stop       停止 Clash 服务")
		fmt.Println("  restart    重启 Clash 服务")
		fmt.Println("  status     检查 Clash 服务状态")
		fmt.Println("  update     更新 Clash Premium")
		fmt.Println("  reset-config     重置配置文件")
		fmt.Println("  proxy      节点配置管理")
		fmt.Println("  version    显示版本信息")
		fmt.Println("  help       显示帮助信息")
	}

	// 检查是否提供了子命令
	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(1)
	}

	// 解析子命令
	switch os.Args[1] {
	case "install":
		installClash()
	case "start":
		startClash()
	case "stop":
		stopClash()
	case "restart":
		restartClash()
	case "status":
		statusClash()
	case "update":
		updateClash()
	case "reset-config":
		generateConfigClash()
	case "proxy":
		manageProxyNodes()
	case "version":
		showVersion()
	case "help":
		flag.Usage()
	default:
		fmt.Printf("未知命令: %s\n", os.Args[1])
		flag.Usage()
		os.Exit(1)
	}
}

// 显示版本信息
func showVersion() {
	fmt.Printf("Clash 管理工具 v%s\n", VERSION)
}

// 安装并配置 Clash Premium
func installClash() {
	fmt.Println("开始自动配置 Clash Premium...")

	// 创建临时工作目录
	workDir, err := createWorkDir()
	if err != nil {
		fmt.Printf("创建工作目录失败: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(workDir)

	// 从内置资源解压 clash-premium-installer
	fmt.Println("提取内置 clash-premium-installer...")
	if err := extractEmbeddedResources(workDir); err != nil {
		fmt.Printf("提取 clash-premium-installer 失败: %v\n", err)
		os.Exit(1)
	}

	// 安装器目录路径
	installerDir := filepath.Join(workDir, "clash-premium-installer")

	// 修改 installer.sh 脚本
	fmt.Println("修改 installer.sh 脚本...")
	installerPath := filepath.Join(installerDir, "installer.sh")
	if err := modifyInstallerScript(installerPath); err != nil {
		fmt.Printf("修改 installer.sh 失败: %v\n", err)
		os.Exit(1)
	}

	// 修改 setup-tun.sh 脚本
	fmt.Println("修改 setup-tun.sh 脚本...")
	tunScriptPath := filepath.Join(installerDir, "scripts", "setup-tun.sh")
	if err := modifyTunScript(tunScriptPath); err != nil {
		fmt.Printf("修改 setup-tun.sh 失败: %v\n", err)
		os.Exit(1)
	}

	// 运行安装脚本
	fmt.Println("安装 Clash Premium...")
	if err := runInstaller(installerDir); err != nil {
		fmt.Printf("安装 Clash Premium 失败: %v\n", err)
		os.Exit(1)
	}

	// 安装内置的 Country.mmdb 文件
	fmt.Println("安装 Country.mmdb 文件...")
	if err := installCountryMMDBFromEmbedded(); err != nil {
		fmt.Printf("安装 Country.mmdb 失败: %v\n", err)
		os.Exit(1)
	}

	// 生成配置文件
	fmt.Println("生成 Clash 配置文件...")
	if err := generateClashConfig(); err != nil {
		fmt.Printf("生成配置文件失败: %v\n", err)
		os.Exit(1)
	}

	// 配置 systemd resolved
	fmt.Println("配置 systemd resolved...")
	if err := configureResolved(); err != nil {
		fmt.Printf("配置 systemd resolved 失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Clash Premium 配置完成！")
}

// 启动 Clash 服务
func startClash() {
	fmt.Println("正在启动 Clash 服务...")
	cmd := exec.Command("systemctl", "start", "clash")
	if err := cmd.Run(); err != nil {
		fmt.Printf("启动 Clash 服务失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Clash 服务已启动")
}

// 停止 Clash 服务
func stopClash() {
	fmt.Println("正在停止 Clash 服务...")
	cmd := exec.Command("systemctl", "stop", "clash")
	if err := cmd.Run(); err != nil {
		fmt.Printf("停止 Clash 服务失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Clash 服务已停止")
}

// 重启 Clash 服务
func restartClash() {
	fmt.Println("正在重启 Clash 服务...")
	cmd := exec.Command("systemctl", "restart", "clash")
	if err := cmd.Run(); err != nil {
		fmt.Printf("重启 Clash 服务失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Clash 服务已重启")
}

// 检查 Clash 服务状态
func statusClash() {
	fmt.Println("检查 Clash 服务状态...")
	cmd := exec.Command("systemctl", "status", "clash")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run() // 忽略退出状态码，因为服务可能未运行
}

// 更新 Clash Premium
func updateClash() {
	fmt.Println("开始更新 Clash Premium...")

	// 创建临时工作目录
	workDir, err := createWorkDir()
	if err != nil {
		fmt.Printf("创建工作目录失败: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(workDir)

	// 从内置资源解压 clash-premium-installer
	fmt.Println("提取内置 clash-premium-installer...")
	if err := extractEmbeddedResources(workDir); err != nil {
		fmt.Printf("提取 clash-premium-installer 失败: %v\n", err)
		os.Exit(1)
	}

	// 安装器目录路径
	installerDir := filepath.Join(workDir, "clash-premium-installer")

	// 修改 installer.sh 脚本
	installerPath := filepath.Join(installerDir, "installer.sh")
	if err := modifyInstallerScript(installerPath); err != nil {
		fmt.Printf("修改 installer.sh 失败: %v\n", err)
		os.Exit(1)
	}

	// 运行更新脚本
	fmt.Println("更新 Clash Premium...")
	cmd := exec.Command("./installer.sh", "update")
	cmd.Dir = installerDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("更新 Clash Premium 失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Clash Premium 更新完成！")
}

// 仅生成或修改配置文件
func generateConfigClash() {
	fmt.Println("生成 Clash 配置文件...")
	if err := generateClashConfig(); err != nil {
		fmt.Printf("生成配置文件失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("配置文件已生成，您可能需要重启 Clash 服务以应用新配置。")
}

func createWorkDir() (string, error) {
	tempDir, err := os.MkdirTemp("", "clash-setup-*")
	if err != nil {
		return "", err
	}
	return tempDir, nil
}

func modifyInstallerScript(filePath string) error {
	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// 替换仓库地址
	modifiedContent := strings.ReplaceAll(string(content), "Dreamacro/clash", "Kuingsmile/clash-core")

	// 写回文件
	return os.WriteFile(filePath, []byte(modifiedContent), 0644)
}

func modifyTunScript(filePath string) error {
	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// 找到需要添加规则的位置并添加规则
	contentStr := string(content)
	
	// 修改 local-dns-redirect 链
	localDnsChain := `    chain local-dns-redirect {
        type nat hook output priority 0; policy accept;
        
        ip protocol != { tcp, udp } accept
        
        meta cgroup $BYPASS_CGROUP_CLASSID accept
        ip daddr 127.0.0.0/8 accept
        ip daddr 10.0.0.0/8 accept
        
        udp dport 53 dnat $FORWARD_DNS_REDIRECT
        tcp dport 53 dnat $FORWARD_DNS_REDIRECT
    }`
	
	// 添加 forward-dns-redirect 链
	forwardDnsChain := `    chain forward-dns-redirect {
        type nat hook prerouting priority 0; policy accept;
        
        ip protocol != { tcp, udp } accept
        ip daddr 10.0.0.0/8 accept
        
        udp dport 53 dnat $FORWARD_DNS_REDIRECT
        tcp dport 53 dnat $FORWARD_DNS_REDIRECT
    }`

	// 替换现有的 local-dns-redirect 链并添加 forward-dns-redirect 链
	oldLocalChain := `    chain local-dns-redirect {
        type nat hook output priority 0; policy accept;
        
        ip protocol != { tcp, udp } accept
        
        meta cgroup $BYPASS_CGROUP_CLASSID accept
        ip daddr 127.0.0.0/8 accept
        
        udp dport 53 dnat $FORWARD_DNS_REDIRECT
        tcp dport 53 dnat $FORWARD_DNS_REDIRECT
    }`

	modifiedContent := strings.Replace(contentStr, oldLocalChain, localDnsChain+"\n\n"+forwardDnsChain, 1)

	// 写回文件
	return os.WriteFile(filePath, []byte(modifiedContent), 0644)
}

func runInstaller(installerDir string) error {
	cmd := exec.Command("./installer.sh", "install")
	cmd.Dir = installerDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func configureResolved() error {
	// 配置 /etc/systemd/resolved.conf
	resolvedConf := `[Resolve]
DNS=127.0.0.1 
FallbackDNS=114.114.114.114 
DNSStubListener=no
`
	err := os.WriteFile("/etc/systemd/resolved.conf", []byte(resolvedConf), 0644)
	if err != nil {
		return err
	}
	
	// 重启 systemd-resolved 服务
	cmd := exec.Command("systemctl", "restart", "systemd-resolved")
	return cmd.Run()
}

func generateClashConfig() error {
	// 创建配置目录
	configDir := "/srv/clash"
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// 获取代理配置
	proxies, err := collectProxyConfigs()
	if err != nil {
		return err
	}

	// 获取代理组配置
	proxyGroup, err := collectProxyGroupConfig(proxies)
	if err != nil {
		return err
	}

	// 读取模板配置
	templateContent, err := readConfigTemplate()
	if err != nil {
		return err
	}

	// 生成代理配置字符串
	proxiesStr := generateProxiesConfig(proxies)
	
	// 生成代理组配置字符串
	proxyGroupStr := generateProxyGroupConfig(proxyGroup)
	
	// 将代理组名称替换到规则中
	templateContent = strings.ReplaceAll(templateContent, "PROXY", proxyGroup.Name)
	
	// 插入代理和代理组配置
	configContent := strings.Replace(templateContent, 
		"# proxies 和 proxy-groups 将根据用户输入自动生成", 
		"proxies:\n" + proxiesStr + "\nproxy-groups:\n" + proxyGroupStr, 1)
	
	// 写入配置文件
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return err
	}

	// 重启 Clash 服务以应用新配置
	cmd := exec.Command("systemctl", "restart", "clash")
	return cmd.Run()
}

func collectProxyConfigs() ([]ProxyConfig, error) {
	var proxies []ProxyConfig
	var proxyCount int
	
	fmt.Println("\n节点配置方式:")
	fmt.Println("1. 手动输入参数")
	fmt.Println("2. 通过协议链接(URL)导入")
	fmt.Println("3. 从现有配置文件导入")
	
	var configMethod int
	fmt.Print("请选择配置方式 (1/2/3): ")
	fmt.Scanln(&configMethod)
	
	if configMethod == 2 {
		// 通过URL导入
		return collectProxyConfigsFromURL()
	} else if configMethod == 3 {
		// 从现有配置文件导入
		return collectProxyConfigsFromFile()
	}
	
	// 原有的手动配置方式
	fmt.Print("请输入需要配置的代理数量: ")
	fmt.Scanln(&proxyCount)
	
	reader := bufio.NewReader(os.Stdin)
	
	for i := 0; i < proxyCount; i++ {
		fmt.Printf("\n== 配置代理 %d ==\n", i+1)
		
		var proxy ProxyConfig
		
		fmt.Print("代理名称: ")
		proxy.Name, _ = reader.ReadString('\n')
		proxy.Name = strings.TrimSpace(proxy.Name)
		
		fmt.Print("代理类型 (vmess/ss/trojan/...): ")
		proxy.Type, _ = reader.ReadString('\n')
		proxy.Type = strings.TrimSpace(proxy.Type)
		
		fmt.Print("服务器地址: ")
		proxy.Server, _ = reader.ReadString('\n')
		proxy.Server = strings.TrimSpace(proxy.Server)
		
		fmt.Print("端口: ")
		proxy.Port, _ = reader.ReadString('\n')
		proxy.Port = strings.TrimSpace(proxy.Port)
		
		if proxy.Type == "vmess" {
			fmt.Print("UUID: ")
			proxy.UUID, _ = reader.ReadString('\n')
			proxy.UUID = strings.TrimSpace(proxy.UUID)
			
			fmt.Print("AlterID (默认为0): ")
			proxy.AlterId, _ = reader.ReadString('\n')
			proxy.AlterId = strings.TrimSpace(proxy.AlterId)
			if proxy.AlterId == "" {
				proxy.AlterId = "0"
			}
			
			fmt.Print("加密方式 (默认为auto): ")
			proxy.Cipher, _ = reader.ReadString('\n')
			proxy.Cipher = strings.TrimSpace(proxy.Cipher)
			if proxy.Cipher == "" {
				proxy.Cipher = "auto"
			}
		} else if proxy.Type == "ss" || proxy.Type == "shadowsocks" {
			fmt.Print("密码: ")
			proxy.Password, _ = reader.ReadString('\n')
			proxy.Password = strings.TrimSpace(proxy.Password)
			
			fmt.Print("加密方式: ")
			proxy.Cipher, _ = reader.ReadString('\n')
			proxy.Cipher = strings.TrimSpace(proxy.Cipher)
		} else if proxy.Type == "trojan" {
			fmt.Print("密码: ")
			proxy.Password, _ = reader.ReadString('\n')
			proxy.Password = strings.TrimSpace(proxy.Password)
		}
		
		proxies = append(proxies, proxy)
	}
	
	return proxies, nil
}

// 通过URL链接导入代理配置
func collectProxyConfigsFromURL() ([]ProxyConfig, error) {
	reader := bufio.NewReader(os.Stdin)
	
	fmt.Print("\n请输入代理URL: ")
	urlStr, _ := reader.ReadString('\n')
	urlStr = strings.TrimSpace(urlStr)
	
	if urlStr == "" {
		return nil, fmt.Errorf("URL不能为空")
	}
	
	// 检查URL格式并解析
	var proxies []ProxyConfig
	
	// 如果URL是单个代理链接
	if strings.HasPrefix(urlStr, "vmess://") {
		proxy, err := parseVMessURL(urlStr)
		if err != nil {
			return nil, err
		}
		proxies = append(proxies, proxy)
	} else if strings.HasPrefix(urlStr, "ss://") {
		proxy, err := parseSSURL(urlStr)
		if err != nil {
			return nil, err
		}
		proxies = append(proxies, proxy)
	} else if strings.HasPrefix(urlStr, "trojan://") {
		proxy, err := parseTrojanURL(urlStr)
		if err != nil {
			return nil, err
		}
		proxies = append(proxies, proxy)
	} else if strings.HasPrefix(urlStr, "http://") || strings.HasPrefix(urlStr, "https://") {
		// 如果是订阅链接，尝试下载并解析
		resp, err := http.Get(urlStr)
		if err != nil {
			return nil, fmt.Errorf("获取订阅内容失败: %v", err)
		}
		defer resp.Body.Close()
		
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("读取订阅内容失败: %v", err)
		}
		
		// 尝试Base64解码
		decodedBody, err := base64.StdEncoding.DecodeString(string(body))
		if err != nil {
			// 如果不是Base64编码，使用原始内容
			decodedBody = body
		}
		
		// 按行分割并解析每一行
		lines := strings.Split(string(decodedBody), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			
			var proxy ProxyConfig
			var err error
			
			if strings.HasPrefix(line, "vmess://") {
				proxy, err = parseVMessURL(line)
			} else if strings.HasPrefix(line, "ss://") {
				proxy, err = parseSSURL(line)
			} else if strings.HasPrefix(line, "trojan://") {
				proxy, err = parseTrojanURL(line)
			} else {
				continue // 跳过不支持的协议
			}
			
			if err != nil {
				fmt.Printf("解析代理失败: %v, 跳过\n", err)
				continue
			}
			
			proxies = append(proxies, proxy)
		}
	} else {
		return nil, fmt.Errorf("不支持的URL格式")
	}
	
	if len(proxies) == 0 {
		return nil, fmt.Errorf("未找到有效的代理")
	}
	
	return proxies, nil
}

// 从现有配置文件导入代理配置
func collectProxyConfigsFromFile() ([]ProxyConfig, error) {
	reader := bufio.NewReader(os.Stdin)
	
	fmt.Print("\n请输入配置文件路径: ")
	filePath, _ := reader.ReadString('\n')
	filePath = strings.TrimSpace(filePath)
	
	if filePath == "" {
		return nil, fmt.Errorf("文件路径不能为空")
	}
	
	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %v", err)
	}
	
	// 尝试按照不同格式解析
	var proxies []ProxyConfig
	
	// 首先尝试按行分割，查找代理URL
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		var proxy ProxyConfig
		var err error
		
		if strings.HasPrefix(line, "vmess://") {
			proxy, err = parseVMessURL(line)
		} else if strings.HasPrefix(line, "ss://") {
			proxy, err = parseSSURL(line)
		} else if strings.HasPrefix(line, "trojan://") {
			proxy, err = parseTrojanURL(line)
		} else {
			continue
		}
		
		if err != nil {
			fmt.Printf("解析代理失败: %v, 跳过\n", err)
			continue
		}
		
		proxies = append(proxies, proxy)
	}
	
	// 如果没有找到代理URL，尝试解析为Clash配置
	if len(proxies) == 0 {
		var config map[string]interface{}
		if err := yaml.Unmarshal(content, &config); err != nil {
			return nil, fmt.Errorf("解析配置文件失败: %v", err)
		}
		
		proxiesInterface, ok := config["proxies"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("未找到代理配置")
		}
		
		for _, proxyInterface := range proxiesInterface {
			proxyMap, ok := proxyInterface.(map[string]interface{})
			if !ok {
				continue
			}
			
			var proxy ProxyConfig
			
			if name, ok := proxyMap["name"].(string); ok {
				proxy.Name = name
			} else {
				continue
			}
			
			if proxyType, ok := proxyMap["type"].(string); ok {
				proxy.Type = proxyType
			} else {
				continue
			}
			
			if server, ok := proxyMap["server"].(string); ok {
				proxy.Server = server
			} else {
				continue
			}
			
			if port, ok := proxyMap["port"].(float64); ok {
				proxy.Port = fmt.Sprintf("%d", int(port))
			} else if port, ok := proxyMap["port"].(int); ok {
				proxy.Port = fmt.Sprintf("%d", port)
			} else if port, ok := proxyMap["port"].(string); ok {
				proxy.Port = port
			} else {
				continue
			}
			
			if proxy.Type == "vmess" {
				if uuid, ok := proxyMap["uuid"].(string); ok {
					proxy.UUID = uuid
				}
				
				if alterId, ok := proxyMap["alterId"].(float64); ok {
					proxy.AlterId = fmt.Sprintf("%d", int(alterId))
				} else if alterId, ok := proxyMap["alterId"].(int); ok {
					proxy.AlterId = fmt.Sprintf("%d", alterId)
				} else if alterId, ok := proxyMap["alterId"].(string); ok {
					proxy.AlterId = alterId
				} else {
					proxy.AlterId = "0"
				}
				
				if cipher, ok := proxyMap["cipher"].(string); ok {
					proxy.Cipher = cipher
				} else {
					proxy.Cipher = "auto"
				}
			} else if proxy.Type == "ss" || proxy.Type == "shadowsocks" {
				if password, ok := proxyMap["password"].(string); ok {
					proxy.Password = password
				}
				
				if cipher, ok := proxyMap["cipher"].(string); ok {
					proxy.Cipher = cipher
				}
			} else if proxy.Type == "trojan" {
				if password, ok := proxyMap["password"].(string); ok {
					proxy.Password = password
				}
			}
			
			proxies = append(proxies, proxy)
		}
	}
	
	if len(proxies) == 0 {
		return nil, fmt.Errorf("未找到有效的代理")
	}
	
	return proxies, nil
}

// 解析代理URL
func parseProxyURL(urlStr string) (ProxyConfig, error) {
	var proxy ProxyConfig
	
	if strings.HasPrefix(urlStr, "ss://") {
		// 解析Shadowsocks URL
		return parseSSURL(urlStr)
	} else if strings.HasPrefix(urlStr, "vmess://") {
		// 解析VMess URL
		return parseVMessURL(urlStr)
	} else if strings.HasPrefix(urlStr, "trojan://") {
		// 解析Trojan URL
		return parseTrojanURL(urlStr)
	}
	
	return proxy, fmt.Errorf("不支持的URL格式")
}

// 解析Shadowsocks URL
func parseSSURL(urlStr string) (ProxyConfig, error) {
	var proxy ProxyConfig
	proxy.Type = "ss"
	
	// ss://BASE64(method:password)@server:port#tag
	// 或 ss://BASE64(method:password@server:port)#tag
	
	// 提取Tag/Name部分
	parts := strings.SplitN(urlStr, "#", 2)
	if len(parts) == 2 {
		proxy.Name = parts[1]
	} else {
		proxy.Name = "Shadowsocks节点"
	}
	
	// 处理主体部分
	mainPart := parts[0][5:] // 去掉 "ss://" 前缀
	
	// 检查是否有@符号来确定编码方式
	if strings.Contains(mainPart, "@") {
		// 格式为 BASE64(method:password)@server:port
		beforeAt := strings.Split(mainPart, "@")[0]
		afterAt := strings.Split(mainPart, "@")[1]
		
		// 解码method:password部分
		decodedBytes, err := decodeBase64UrlSafe(beforeAt)
		if err != nil {
			return proxy, fmt.Errorf("解码失败: %v", err)
		}
		methodPass := string(decodedBytes)
		
		// 分割method和password
		methodPassParts := strings.SplitN(methodPass, ":", 2)
		if len(methodPassParts) != 2 {
			return proxy, fmt.Errorf("无效的编码格式")
		}
		
		proxy.Cipher = methodPassParts[0]
		proxy.Password = methodPassParts[1]
		
		// 解析服务器和端口
		serverPortParts := strings.Split(afterAt, ":")
		if len(serverPortParts) != 2 {
			return proxy, fmt.Errorf("无效的服务器:端口格式")
		}
		
		proxy.Server = serverPortParts[0]
		proxy.Port = serverPortParts[1]
	} else {
		// 整个部分都被BASE64编码
		decodedBytes, err := decodeBase64UrlSafe(mainPart)
		if err != nil {
			return proxy, fmt.Errorf("解码失败: %v", err)
		}
		
		decoded := string(decodedBytes)
		if !strings.Contains(decoded, "@") {
			return proxy, fmt.Errorf("解码后的内容无效")
		}
		
		beforeAt := strings.Split(decoded, "@")[0]
		afterAt := strings.Split(decoded, "@")[1]
		
		// 分割method和password
		methodPassParts := strings.SplitN(beforeAt, ":", 2)
		if len(methodPassParts) != 2 {
			return proxy, fmt.Errorf("无效的编码格式")
		}
		
		proxy.Cipher = methodPassParts[0]
		proxy.Password = methodPassParts[1]
		
		// 解析服务器和端口
		serverPortParts := strings.Split(afterAt, ":")
		if len(serverPortParts) != 2 {
			return proxy, fmt.Errorf("无效的服务器:端口格式")
		}
		
		proxy.Server = serverPortParts[0]
		proxy.Port = serverPortParts[1]
	}
	
	if proxy.Name == "" {
		proxy.Name = proxy.Server
	}
	
	return proxy, nil
}

// 解析VMess URL
func parseVMessURL(urlStr string) (ProxyConfig, error) {
	var proxy ProxyConfig
	proxy.Type = "vmess"
	
	// 移除 "vmess://" 前缀
	base64Str := urlStr[8:]
	
	// 解码Base64内容
	jsonBytes, err := decodeBase64UrlSafe(base64Str)
	if err != nil {
		return proxy, fmt.Errorf("解码VMess URL失败: %v", err)
	}
	
	// 解析JSON
	var vmessConfig map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &vmessConfig); err != nil {
		return proxy, fmt.Errorf("解析VMess配置失败: %v", err)
	}
	
	// 提取配置信息
	if v, ok := vmessConfig["ps"].(string); ok {
		proxy.Name = v
	} else if v, ok := vmessConfig["remarks"].(string); ok {
		proxy.Name = v
	} else {
		proxy.Name = "VMess节点"
	}
	
	if v, ok := vmessConfig["add"].(string); ok {
		proxy.Server = v
	} else if v, ok := vmessConfig["host"].(string); ok {
		proxy.Server = v
	} else {
		return proxy, fmt.Errorf("无法提取服务器地址")
	}
	
	if v, ok := vmessConfig["port"].(float64); ok {
		proxy.Port = fmt.Sprintf("%d", int(v))
	} else if v, ok := vmessConfig["port"].(string); ok {
		proxy.Port = v
	} else {
		return proxy, fmt.Errorf("无法提取端口")
	}
	
	if v, ok := vmessConfig["id"].(string); ok {
		proxy.UUID = v
	} else {
		return proxy, fmt.Errorf("无法提取UUID")
	}
	
	if v, ok := vmessConfig["aid"].(float64); ok {
		proxy.AlterId = fmt.Sprintf("%d", int(v))
	} else if v, ok := vmessConfig["aid"].(string); ok {
		proxy.AlterId = v
	} else {
		proxy.AlterId = "0"
	}
	
	if v, ok := vmessConfig["security"].(string); ok {
		proxy.Cipher = v
	} else {
		proxy.Cipher = "auto"
	}
	
	return proxy, nil
}

// 解析Trojan URL
func parseTrojanURL(urlStr string) (ProxyConfig, error) {
	var proxy ProxyConfig
	proxy.Type = "trojan"
	
	// trojan://password@server:port?allowInsecure=1#name
	u, err := url.Parse(urlStr)
	if err != nil {
		return proxy, fmt.Errorf("解析Trojan URL失败: %v", err)
	}
	
	// 提取密码
	if u.User != nil {
		proxy.Password = u.User.String()
	} else {
		return proxy, fmt.Errorf("无法提取密码")
	}
	
	// 提取服务器和端口
	hostParts := strings.Split(u.Host, ":")
	if len(hostParts) != 2 {
		return proxy, fmt.Errorf("无效的服务器:端口格式")
	}
	
	proxy.Server = hostParts[0]
	proxy.Port = hostParts[1]
	
	// 提取节点名称
	if u.Fragment != "" {
		proxy.Name = u.Fragment
	} else {
		proxy.Name = "Trojan节点"
	}
	
	return proxy, nil
}

// 解码URL安全的Base64内容
func decodeBase64UrlSafe(s string) ([]byte, error) {
	// 替换URL安全字符
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")
	
	// 添加缺失的=号
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	
	return base64.StdEncoding.DecodeString(s)
}

func collectProxyGroupConfig(proxies []ProxyConfig) (ProxyGroupConfig, error) {
	var group ProxyGroupConfig
	reader := bufio.NewReader(os.Stdin)
	
	fmt.Println("\n== 配置代理组 ==")
	
	fmt.Print("代理组名称: ")
	group.Name, _ = reader.ReadString('\n')
	group.Name = strings.TrimSpace(group.Name)
	
	fmt.Print("代理组类型 (select/url-test/fallback/load-balance): ")
	group.Type, _ = reader.ReadString('\n')
	group.Type = strings.TrimSpace(group.Type)
	if group.Type == "" {
		group.Type = "select"
	}
	
	// 收集所有代理名称
	for _, proxy := range proxies {
		group.ProxyNames = append(group.ProxyNames, proxy.Name)
	}
	
	if len(proxies) > 0 && group.Type == "select" {
		fmt.Println("可用的代理:")
		for i, name := range group.ProxyNames {
			fmt.Printf("%d. %s\n", i+1, name)
		}
		
		var selectedIndex int
		fmt.Print("请选择默认代理 (输入序号): ")
		fmt.Scanln(&selectedIndex)
		
		if selectedIndex > 0 && selectedIndex <= len(group.ProxyNames) {
			group.SelectedProxy = group.ProxyNames[selectedIndex-1]
		} else if len(group.ProxyNames) > 0 {
			group.SelectedProxy = group.ProxyNames[0]
		}
	}
	
	return group, nil
}

func generateProxiesConfig(proxies []ProxyConfig) string {
	var result strings.Builder
	
	for _, proxy := range proxies {
		result.WriteString("    - { name: '")
		result.WriteString(proxy.Name)
		result.WriteString("', type: ")
		result.WriteString(proxy.Type)
		result.WriteString(", server: ")
		result.WriteString(proxy.Server)
		result.WriteString(", port: ")
		result.WriteString(proxy.Port)
		
		if proxy.Type == "vmess" {
			result.WriteString(", uuid: ")
			result.WriteString(proxy.UUID)
			result.WriteString(", alterId: ")
			result.WriteString(proxy.AlterId)
			result.WriteString(", cipher: ")
			result.WriteString(proxy.Cipher)
			result.WriteString(", udp: true")
		} else if proxy.Type == "ss" || proxy.Type == "shadowsocks" {
			result.WriteString(", password: ")
			result.WriteString(proxy.Password)
			result.WriteString(", cipher: ")
			result.WriteString(proxy.Cipher)
			result.WriteString(", udp: true")
		} else if proxy.Type == "trojan" {
			result.WriteString(", password: ")
			result.WriteString(proxy.Password)
			result.WriteString(", udp: true")
		}
		
		result.WriteString(" }\n")
	}
	
	return result.String()
}

func generateProxyGroupConfig(group ProxyGroupConfig) string {
	var sb strings.Builder
	
	sb.WriteString("  - name: ")
	sb.WriteString(group.Name)
	sb.WriteString("\n    type: ")
	sb.WriteString(group.Type)
	sb.WriteString("\n    proxies:\n")
	
	// 添加所有代理节点，保持原来的引号格式
	for _, proxyName := range group.ProxyNames {
		sb.WriteString("      - \"")
		sb.WriteString(proxyName)
		sb.WriteString("\"\n")
	}
	
	// 添加内置策略 DIRECT，使用与其他代理相同的格式
	sb.WriteString("      - \"DIRECT\"\n")
	
	// 如果有选中的代理，添加 selected 字段
	if group.Type == "select" && group.SelectedProxy != "" {
		sb.WriteString("    selected: \"")
		sb.WriteString(group.SelectedProxy)
		sb.WriteString("\"\n")
	}
	
	return sb.String()
}

func readConfigTemplate() (string, error) {
	// 从嵌入资源中读取模板
	templateContent, err := readEmbeddedFile("resources/config-template.yaml")
	if err != nil {
		// 如果嵌入资源读取失败，尝试从文件系统读取作为备选
		templateContent, err = os.ReadFile("resources/config-template.yaml")
		if err != nil {
			return "", err
		}
	}
	
	return string(templateContent), nil
}

// 安装内置的 Country.mmdb 文件
func installCountryMMDBFromEmbedded() error {
	// 确保配置目录存在
	configDir := "/etc/clash"
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("创建Clash配置目录失败: %v", err)
	}
	
	// 从嵌入资源中读取Country.mmdb文件
	mmdbContent, err := readEmbeddedFile("resources/Country.mmdb")
	if err != nil {
		return fmt.Errorf("读取嵌入的Country.mmdb文件失败: %v", err)
	}
	
	// 写入到标准位置
	mmdbPath := filepath.Join(configDir, "Country.mmdb")
	if err := os.WriteFile(mmdbPath, mmdbContent, 0644); err != nil {
		return fmt.Errorf("写入Country.mmdb文件失败: %v", err)
	}
	
	// 创建从标准位置到Clash工作目录的符号链接
	clashDir := "/srv/clash"
	if err := os.MkdirAll(clashDir, 0755); err != nil {
		return fmt.Errorf("创建Clash工作目录失败: %v", err)
	}
	
	clashMMDBPath := filepath.Join(clashDir, "Country.mmdb")
	// 如果已存在，先删除
	if _, err := os.Stat(clashMMDBPath); err == nil {
		os.Remove(clashMMDBPath)
	}
	
	// 创建符号链接
	if err := os.Symlink(mmdbPath, clashMMDBPath); err != nil {
		return fmt.Errorf("创建Country.mmdb符号链接失败: %v", err)
	}
	
	// 添加一个到/root/.config/clash/Country.mmdb的符号链接
	rootConfigDir := "/root/.config/clash"
	if err := os.MkdirAll(rootConfigDir, 0755); err != nil {
		return fmt.Errorf("创建root用户Clash配置目录失败: %v", err)
	}
	
	rootMMDBPath := filepath.Join(rootConfigDir, "Country.mmdb")
	// 如果已存在，先删除
	if _, err := os.Stat(rootMMDBPath); err == nil {
		os.Remove(rootMMDBPath)
	}
	
	// 创建符号链接
	if err := os.Symlink(mmdbPath, rootMMDBPath); err != nil {
		return fmt.Errorf("创建到root用户目录的Country.mmdb符号链接失败: %v", err)
	}
	
	// 设置GEOIP_DATABASE环境变量到systemd服务
	// 创建systemd环境变量目录
	envDir := "/etc/systemd/system/clash.service.d"
	if err := os.MkdirAll(envDir, 0755); err != nil {
		return fmt.Errorf("创建systemd服务环境变量目录失败: %v", err)
	}
	
	// 创建环境变量配置文件
	envContent := `[Service]
Environment="GEOIP_DATABASE=` + mmdbPath + `"
`
	envPath := filepath.Join(envDir, "environment.conf")
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		return fmt.Errorf("写入systemd环境变量配置失败: %v", err)
	}
	
	// 重新加载systemd配置
	reloadCmd := exec.Command("systemctl", "daemon-reload")
	if err := reloadCmd.Run(); err != nil {
		return fmt.Errorf("重新加载systemd配置失败: %v", err)
	}
	
	fmt.Printf("Country.mmdb已安装到%s，并创建了符号链接到%s和%s\n", 
		mmdbPath, clashMMDBPath, rootMMDBPath)
	fmt.Printf("已设置GEOIP_DATABASE环境变量到%s\n", mmdbPath)
	return nil
}

// 读取Clash配置文件
func readClashConfig() (map[string]interface{}, error) {
	configPath := "/srv/clash/config.yaml"
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	
	var config map[string]interface{}
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, err
	}
	
	return config, nil
}

// 保存Clash配置文件
func saveClashConfig(config map[string]interface{}) error {
	configPath := "/srv/clash/config.yaml"
	content, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	
	return os.WriteFile(configPath, content, 0644)
}
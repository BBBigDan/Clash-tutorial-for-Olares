package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
	"os/exec"
	"sort"
)

// 使用与 main.go 兼容的结构体，或者完全移除此定义
// 注释掉 ProxyConfig 的重复声明
/*
type ProxyConfig struct {
	Name     string
	Type     string
	Server   string
	Port     string
	Password string
	Cipher   string
	UUID     string
	AlterId  string
	SNI      string
}
*/

// 代表当前选中的代理组和代理
type SelectedProxyInfo struct {
	GroupName     string
	SelectedProxy string
}

// 管理代理节点配置的实现函数
func manageProxyNodes() {
	// 创建交互式菜单
	for {
		clearScreen()
		fmt.Println("===== Clash 代理节点管理 =====")
		fmt.Println("1. 查看所有节点")
		fmt.Println("2. 添加新节点")
		fmt.Println("3. 删除节点")
		fmt.Println("4. 导入节点")
		fmt.Println("5. 查看节点状态和连接速度")
		fmt.Println("6. 切换使用的节点")
		fmt.Println("7. 输出配置文件内容")
		fmt.Println("0. 返回主菜单")
		fmt.Println("=============================")
		
		var choice int
		fmt.Print("请选择操作 [0-7]: ")
		fmt.Scanln(&choice)
		
		switch choice {
		case 1:
			interactiveListProxies()
		case 2:
			interactiveAddProxy()
		case 3:
			interactiveDeleteProxy()
		case 4:
			interactiveImportProxies()
		case 5:
			interactiveShowProxyStatus()
		case 6:
			interactiveSelectProxy()
		case 7:
			interactiveShowConfigContent()
		case 0:
			fmt.Println("正在返回主菜单...")
			return
		default:
			fmt.Println("无效的选择，请重试")
			time.Sleep(1 * time.Second)
		}
	}
}

// 清除屏幕的实现函数
func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

// 等待用户按下任意键继续的实现函数
func waitForKeyPress() {
	fmt.Print("\n按回车键继续...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

// 交互式列出所有节点
func interactiveListProxies() {
	clearScreen()
	fmt.Println("===== 所有代理节点 =====")
	
	// 读取配置文件
	config, err := readClashConfig()
	if err != nil {
		fmt.Printf("读取配置文件失败: %v\n", err)
		waitForKeyPress()
		return
	}
	
	// 获取代理节点
	proxies, ok := config["proxies"].([]interface{})
	if !ok {
		fmt.Println("配置文件中未找到代理节点或格式错误")
		waitForKeyPress()
		return
	}
	
	if len(proxies) == 0 {
		fmt.Println("没有配置任何代理节点")
		waitForKeyPress()
		return
	}
	
	// 打印节点信息
	fmt.Printf("共找到 %d 个节点:\n\n", len(proxies))
	
	for i, proxyInterface := range proxies {
		proxy, ok := proxyInterface.(map[string]interface{})
		if !ok {
			continue
		}
		
		name := proxy["name"]
		proxyType := proxy["type"]
		server := proxy["server"]
		port := proxy["port"]
		
		fmt.Printf("%d. 名称: %v\n", i+1, name)
		fmt.Printf("   类型: %v\n", proxyType)
		fmt.Printf("   服务器: %v\n", server)
		fmt.Printf("   端口: %v\n", port)
		
		// 根据类型显示特定信息
		if proxyType == "vmess" {
			if uuid, ok := proxy["uuid"].(string); ok {
				fmt.Printf("   UUID: %s\n", uuid)
			}
			if aid, ok := proxy["alterId"].(string); ok {
				fmt.Printf("   AlterId: %s\n", aid)
			}
		} else if proxyType == "ss" || proxyType == "shadowsocks" {
			if cipher, ok := proxy["cipher"].(string); ok {
				fmt.Printf("   加密方式: %s\n", cipher)
			}
		}
		
		fmt.Println("   ------------------------")
	}
	
	waitForKeyPress()
}

// 交互式添加新节点
func interactiveAddProxy() {
	clearScreen()
	fmt.Println("===== 添加新代理节点 =====")
	
	// 读取配置文件
	config, err := readClashConfig()
	if err != nil {
		fmt.Printf("读取配置文件失败: %v\n", err)
		waitForKeyPress()
		return
	}
	
	// 收集代理配置
	proxy, err := collectSingleProxyConfig()
	if err != nil {
		fmt.Printf("收集代理配置失败: %v\n", err)
		waitForKeyPress()
		return
	}
	
	// 获取当前代理列表
	var proxies []interface{}
	if existingProxies, ok := config["proxies"].([]interface{}); ok {
		proxies = existingProxies
	}
	
	// 检查节点名称是否已存在
	for _, p := range proxies {
		if proxyMap, ok := p.(map[string]interface{}); ok {
			if name, ok := proxyMap["name"].(string); ok && name == proxy.Name {
				fmt.Printf("节点名称 '%s' 已存在，请使用不同的名称\n", proxy.Name)
				waitForKeyPress()
				return
			}
		}
	}
	
	// 创建新的代理配置项
	proxyMap := make(map[string]interface{})
	proxyMap["name"] = proxy.Name
	proxyMap["type"] = proxy.Type
	proxyMap["server"] = proxy.Server
	proxyMap["port"] = proxy.Port
	
	// 添加特定类型的配置
	switch proxy.Type {
	case "ss":
		proxyMap["cipher"] = proxy.Cipher
		proxyMap["password"] = proxy.Password
	case "vmess":
		proxyMap["uuid"] = proxy.UUID
		proxyMap["alterId"] = proxy.AlterId
		proxyMap["cipher"] = proxy.Cipher
	case "trojan":
		proxyMap["password"] = proxy.Password
		if proxy.SNI != "" {
			proxyMap["sni"] = proxy.SNI
		}
	}
	
	// 添加共用选项
	proxyMap["udp"] = true
	
	// 添加到代理列表
	proxies = append(proxies, proxyMap)
	config["proxies"] = proxies
	
	// 更新代理组
	updateProxyGroup(config, proxy.Name)
	
	// 保存配置
	if err := saveClashConfig(config); err != nil {
		fmt.Printf("保存配置文件失败: %v\n", err)
		waitForKeyPress()
		return
	}
	
	fmt.Printf("\n代理节点 '%s' 已添加！\n", proxy.Name)
	fmt.Println("您需要重启 Clash 服务以应用更改。现在重启吗？[y/n]")
	
	var restart string
	fmt.Scanln(&restart)
	if strings.ToLower(restart) == "y" {
		cmd := exec.Command("systemctl", "restart", "clash")
		if err := cmd.Run(); err != nil {
			fmt.Printf("重启 Clash 服务失败: %v\n", err)
		} else {
			fmt.Println("Clash 服务已重启")
		}
	}
	
	waitForKeyPress()
}

// 交互式删除节点
func interactiveDeleteProxy() {
	clearScreen()
	fmt.Println("===== 删除代理节点 =====")
	
	// 读取配置文件
	config, err := readClashConfig()
	if err != nil {
		fmt.Printf("读取配置文件失败: %v\n", err)
		waitForKeyPress()
		return
	}
	
	// 获取代理节点
	proxies, ok := config["proxies"].([]interface{})
	if !ok {
		fmt.Println("配置文件中未找到代理节点或格式错误")
		waitForKeyPress()
		return
	}
	
	if len(proxies) == 0 {
		fmt.Println("没有配置任何代理节点")
		waitForKeyPress()
		return
	}
	
	// 显示所有节点
	fmt.Println("可用节点列表:")
	var proxyNames []string
	for i, proxyInterface := range proxies {
		proxy, ok := proxyInterface.(map[string]interface{})
		if !ok {
			continue
		}
		
		name, ok := proxy["name"].(string)
		if !ok {
			continue
		}
		
		proxyNames = append(proxyNames, name)
		fmt.Printf("%d. %s\n", i+1, name)
	}
	
	// 选择要删除的节点
	var choice int
	fmt.Print("\n请选择要删除的节点 (输入序号，0取消): ")
	fmt.Scanln(&choice)
	
	if choice == 0 {
		fmt.Println("操作已取消")
		waitForKeyPress()
		return
	}
	
	if choice < 1 || choice > len(proxyNames) {
		fmt.Println("无效的选择")
		waitForKeyPress()
		return
	}
	
	// 获取要删除的节点名称
	proxyToDelete := proxyNames[choice-1]
	
	// 从代理列表中移除
	var newProxies []interface{}
	for _, proxyInterface := range proxies {
		proxy, ok := proxyInterface.(map[string]interface{})
		if !ok {
			continue
		}
		
		name, ok := proxy["name"].(string)
		if !ok || name == proxyToDelete {
			continue
		}
		
		newProxies = append(newProxies, proxyInterface)
	}
	
	// 更新配置
	config["proxies"] = newProxies
	
	// 从代理组中移除
	removeFromProxyGroups(config, proxyToDelete)
	
	// 保存配置
	if err := saveClashConfig(config); err != nil {
		fmt.Printf("保存配置文件失败: %v\n", err)
		waitForKeyPress()
		return
	}
	
	fmt.Printf("\n代理节点 '%s' 已删除！\n", proxyToDelete)
	fmt.Println("您需要重启 Clash 服务以应用更改。现在重启吗？[y/n]")
	
	var restart string
	fmt.Scanln(&restart)
	if strings.ToLower(restart) == "y" {
		cmd := exec.Command("systemctl", "restart", "clash")
		if err := cmd.Run(); err != nil {
			fmt.Printf("重启 Clash 服务失败: %v\n", err)
		} else {
			fmt.Println("Clash 服务已重启")
		}
	}
	
	waitForKeyPress()
}

// 收集单个代理配置信息
func collectSingleProxyConfig() (ProxyConfig, error) {
	var config ProxyConfig
	reader := bufio.NewReader(os.Stdin)
	
	// 询问代理类型
	fmt.Println("\n请选择代理类型:")
	fmt.Println("1. Shadowsocks (SS)")
	fmt.Println("2. VMess")
	fmt.Println("3. Trojan")
	
	var typeChoice int
	fmt.Print("请选择 [1-3]: ")
	fmt.Scanln(&typeChoice)
	
	switch typeChoice {
	case 1:
		config.Type = "ss"
	case 2:
		config.Type = "vmess"
	case 3:
		config.Type = "trojan"
	default:
		return config, fmt.Errorf("无效的代理类型选择")
	}
	
	// 询问名称
	fmt.Print("\n请输入节点名称: ")
	name, _ := reader.ReadString('\n')
	config.Name = strings.TrimSpace(name)
	
	// 询问服务器地址
	fmt.Print("请输入服务器地址: ")
	server, _ := reader.ReadString('\n')
	config.Server = strings.TrimSpace(server)
	
	// 询问端口
	fmt.Print("请输入端口: ")
	port, _ := reader.ReadString('\n')
	config.Port = strings.TrimSpace(port)
	
	// 根据类型询问特定信息
	switch config.Type {
	case "ss":
		fmt.Print("请输入加密方法(默认为aes-256-gcm): ")
		cipher, _ := reader.ReadString('\n')
		cipher = strings.TrimSpace(cipher)
		if cipher == "" {
			cipher = "aes-256-gcm"
		}
		config.Cipher = cipher
		
		fmt.Print("请输入密码: ")
		password, _ := reader.ReadString('\n')
		config.Password = strings.TrimSpace(password)
		
	case "vmess":
		fmt.Print("请输入UUID: ")
		uuid, _ := reader.ReadString('\n')
		config.UUID = strings.TrimSpace(uuid)
		
		fmt.Print("请输入alterId(默认为0): ")
		alterId, _ := reader.ReadString('\n')
		alterId = strings.TrimSpace(alterId)
		if alterId == "" {
			alterId = "0"
		}
		config.AlterId = alterId
		
		fmt.Print("请输入加密方法(默认为auto): ")
		cipher, _ := reader.ReadString('\n')
		cipher = strings.TrimSpace(cipher)
		if cipher == "" {
			cipher = "auto"
		}
		config.Cipher = cipher
		
	case "trojan":
		fmt.Print("请输入密码: ")
		password, _ := reader.ReadString('\n')
		config.Password = strings.TrimSpace(password)
		
		fmt.Print("请输入SNI(可选): ")
		sni, _ := reader.ReadString('\n')
		config.SNI = strings.TrimSpace(sni)
	}
	
	// 检查必要信息
	if config.Name == "" {
		// 如果没有提供名称，使用服务器和端口作为名称
		config.Name = fmt.Sprintf("%s-%s:%s", strings.ToUpper(config.Type), config.Server, config.Port)
	}
	
	if config.Server == "" || config.Port == "" {
		return config, fmt.Errorf("服务器地址和端口是必需的")
	}
	
	// 根据代理类型检查特定信息
	switch config.Type {
	case "ss":
		if config.Password == "" {
			return config, fmt.Errorf("密码是必需的")
		}
	case "vmess":
		if config.UUID == "" {
			return config, fmt.Errorf("UUID是必需的")
		}
	case "trojan":
		if config.Password == "" {
			return config, fmt.Errorf("密码是必需的")
		}
	}
	
	return config, nil
}

// 交互式选择代理
func interactiveSelectProxy() {
	clearScreen()
	fmt.Println("===== 切换代理节点 =====")
	
	// 读取配置文件
	config, err := readClashConfig()
	if err != nil {
		fmt.Printf("读取配置文件失败: %v\n", err)
		waitForKeyPress()
		return
	}
	
	// 获取代理节点
	proxiesInterface, ok := config["proxies"].([]interface{})
	if !ok {
		fmt.Println("配置文件中未找到代理节点或格式错误")
		waitForKeyPress()
		return
	}
	
	// 转换为 ProxyConfig 数组
	var proxies []ProxyConfig
	for _, p := range proxiesInterface {
		if proxy, ok := p.(map[string]interface{}); ok {
			name, _ := proxy["name"].(string)
			proxyType, _ := proxy["type"].(string)
			server, _ := proxy["server"].(string)
			port, _ := proxy["port"].(string)
			
			proxyConfig := ProxyConfig{
				Name: name,
				Type: proxyType,
				Server: server,
				Port: port,
			}
			
			proxies = append(proxies, proxyConfig)
		}
	}
	
	// 获取当前选中的代理组
	selInfo, selErr := getSelectedProxy()
	if selErr == nil {
		fmt.Printf("\n当前选中的代理组: %s\n", selInfo.GroupName)
		fmt.Printf("当前使用的节点: %s\n", selInfo.SelectedProxy)
	}
	
	// 询问是否需要先测速再选择节点
	fmt.Print("\n是否需要先测速再选择节点？这将帮助您选择最快的节点 [y/n]: ")
	var testSpeed string
	fmt.Scanln(&testSpeed)
	
	var proxyList []string
	var delayResults map[string]int
	
	// 用于存储代理名称和对应的延迟，方便排序
	type ProxyDelay struct {
		Name  string
		Delay int
	}
	var proxyDelays []ProxyDelay
	
	// 如果用户选择测速
	if strings.ToLower(testSpeed) == "y" {
		// 准备代理名称列表
		for _, p := range proxies {
			proxyList = append(proxyList, p.Name)
		}
		
		// 选择测速方式
		fmt.Println("\n请选择测速方式:")
		fmt.Println("1. 使用 Clash API 测速")
		fmt.Println("2. 使用简化方法测速 (直接测试节点连接)")
		fmt.Println("3. 使用可靠方法测速 (推荐)")
		
		var speedTestMethod int
		fmt.Print("请选择 [1-3]: ")
		fmt.Scanln(&speedTestMethod)
		
		// 询问是否使用直连模式测速
		fmt.Print("\n使用直连模式测速可能会提高准确性，但会临时断开当前连接。使用直连模式？[y/n]: ")
		var useDirect string
		fmt.Scanln(&useDirect)
		useDirectMode := strings.ToLower(useDirect) == "y"
		
		fmt.Println("\n开始测试节点延迟，请稍候...")
		
		var delayErr error
		
		switch speedTestMethod {
		case 1:
			delayResults, delayErr = getProxyDelays(proxyList, useDirectMode)
		case 2:
			delayResults, delayErr = getProxyDelaysSimple(proxyList)
		case 3:
			delayResults, delayErr = getProxyDelaysReliable(proxyList)
		default:
			delayResults, delayErr = getProxyDelaysSimple(proxyList)
		}
		
		if delayErr != nil {
			fmt.Printf("\n获取代理延迟信息失败: %v\n", delayErr)
		} else {
			// 将结果转换为可排序的结构
			for name, delay := range delayResults {
				proxyDelays = append(proxyDelays, ProxyDelay{Name: name, Delay: delay})
			}
			
			// 按延迟排序，不可连接的放最后
			sort.Slice(proxyDelays, func(i, j int) bool {
				// 如果某个代理不可连接（延迟为负数），放到最后
				if proxyDelays[i].Delay < 0 {
					return false
				}
				if proxyDelays[j].Delay < 0 {
					return true
				}
				// 否则按延迟从小到大排序
				return proxyDelays[i].Delay < proxyDelays[j].Delay
			})
			
			// 显示测速结果
			fmt.Println("\n节点测速结果（按延迟排序）:")
			fmt.Println("----------------------------")
			
			for i, pd := range proxyDelays {
				if pd.Delay > 0 {
					fmt.Printf("%d. %s - 延迟: %d ms\n", i+1, pd.Name, pd.Delay)
				} else {
					fmt.Printf("%d. %s - 无法连接\n", i+1, pd.Name)
				}
			}
			fmt.Println("----------------------------")
		}
	} else {
		// 如果不测速，按原始顺序显示
		fmt.Println("\n可用的代理节点:")
		for i, proxy := range proxies {
			fmt.Printf("%d. %s (%s:%s)\n", i+1, proxy.Name, proxy.Server, proxy.Port)
		}
	}
	
	// 让用户选择
	var choice int
	fmt.Print("\n请选择要使用的节点 (0取消): ")
	fmt.Scanln(&choice)
	
	if choice == 0 {
		fmt.Println("操作已取消")
		waitForKeyPress()
		return
	}
	
	var selectedProxy string
	
	// 根据是否测速决定如何获取选择的代理
	if strings.ToLower(testSpeed) == "y" && len(proxyDelays) > 0 {
		if choice <= 0 || choice > len(proxyDelays) {
			fmt.Println("无效的选择")
			waitForKeyPress()
			return
		}
		selectedProxy = proxyDelays[choice-1].Name
	} else {
		if choice <= 0 || choice > len(proxies) {
			fmt.Println("无效的选择")
			waitForKeyPress()
			return
		}
		selectedProxy = proxies[choice-1].Name
	}
	
	// 如果获取不到当前的代理组，要求用户输入
	var groupName string
	if selErr != nil {
		fmt.Print("\n请输入要设置的代理组名称: ")
		fmt.Scanln(&groupName)
	} else {
		groupName = selInfo.GroupName
	}
	
	// 切换代理
	if err := switchProxy(groupName, selectedProxy); err != nil {
		fmt.Printf("切换代理失败: %v\n", err)
	} else {
		fmt.Printf("已将代理组 %s 切换到 %s\n", groupName, selectedProxy)
	}
	
	waitForKeyPress()
}

// 交互式显示配置文件内容
func interactiveShowConfigContent() {
	clearScreen()
	fmt.Println("===== Clash 配置文件内容 =====")
	
	// 获取配置文件路径
	configPath := getDefaultConfigPath()
	if configPath == "" {
		fmt.Println("未找到配置文件路径")
		waitForKeyPress()
		return
	}
	
	// 读取配置文件内容
	content, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("读取配置文件失败: %v\n", err)
		waitForKeyPress()
		return
	}
	
	// 显示配置文件内容
	fmt.Println("\n--- 配置文件内容开始 ---")
	fmt.Println(string(content))
	fmt.Println("--- 配置文件内容结束 ---")
	
	// 询问是否要保存内容到文件
	fmt.Print("\n是否要将配置文件内容保存到单独的文件？[y/n]: ")
	var saveChoice string
	fmt.Scanln(&saveChoice)
	
	if strings.ToLower(saveChoice) == "y" {
		// 询问保存路径
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("请输入保存路径(默认为./clash_config_backup.yaml): ")
		savePath, _ := reader.ReadString('\n')
		savePath = strings.TrimSpace(savePath)
		
		if savePath == "" {
			savePath = "./clash_config_backup.yaml"
		}
		
		// 保存文件
		err := os.WriteFile(savePath, content, 0644)
		if err != nil {
			fmt.Printf("保存文件失败: %v\n", err)
		} else {
			fmt.Printf("配置文件已保存到 %s\n", savePath)
		}
	}
	
	waitForKeyPress()
}

// 获取默认配置文件路径
func getDefaultConfigPath() string {
	// 尝试从环境变量获取
	if path := os.Getenv("CLASH_CONFIG_PATH"); path != "" {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	
	// 常见的配置文件位置
	commonPaths := []string{
		"/srv/clash/config.yaml",
		"/etc/clash/config.yaml",
		"/usr/local/etc/clash/config.yaml",
		"$HOME/.config/clash/config.yaml",
		"./config.yaml",
	}
	
	// 替换$HOME为用户主目录
	homeDir, err := os.UserHomeDir()
	if err == nil {
		for i, path := range commonPaths {
			commonPaths[i] = strings.Replace(path, "$HOME", homeDir, 1)
		}
	}
	
	// 检查文件是否存在
	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	
	return ""
} 
package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"
	"gopkg.in/yaml.v3"
)

// 交互式导入节点
func interactiveImportProxies() {
	clearScreen()
	fmt.Println("===== 导入代理节点 =====")
	fmt.Println("1. 从订阅链接导入")
	fmt.Println("2. 从Base64编码字符串导入")
	fmt.Println("3. 从节点链接(URI)导入")
	fmt.Println("4. 从YAML文件导入")
	fmt.Println("0. 返回")
	
	var choice int
	fmt.Print("\n请选择导入方式 [0-4]: ")
	fmt.Scanln(&choice)
	
	switch choice {
	case 1:
		importFromSubscription()
	case 2:
		importFromBase64()
	case 3:
		importFromNodeURIs()
	case 4:
		importFromYAML()
	case 0:
		return
	default:
		fmt.Println("无效的选择")
		time.Sleep(1 * time.Second)
	}
}

// 从订阅链接导入
func importFromSubscription() {
	clearScreen()
	fmt.Println("===== 从订阅链接导入 =====")
	
	// 获取订阅链接
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("请输入订阅链接: ")
	subURL, _ := reader.ReadString('\n')
	subURL = strings.TrimSpace(subURL)
	
	if subURL == "" {
		fmt.Println("订阅链接不能为空")
		waitForKeyPress()
		return
	}
	
	// 发送HTTP请求获取订阅内容
	fmt.Println("正在获取订阅内容...")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(subURL)
	if err != nil {
		fmt.Printf("获取订阅内容失败: %v\n", err)
		waitForKeyPress()
		return
	}
	defer resp.Body.Close()
	
	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("读取订阅内容失败: %v\n", err)
		waitForKeyPress()
		return
	}
	
	// 解码Base64内容
	decodedBody, err := base64.StdEncoding.DecodeString(string(body))
	if err != nil {
		fmt.Printf("解码订阅内容失败，尝试作为普通文本处理: %v\n", err)
		decodedBody = body
	}
	
	// 处理URI列表
	uriList := strings.Split(string(decodedBody), "\n")
	var validURIs []string
	for _, uri := range uriList {
		uri = strings.TrimSpace(uri)
		if uri != "" {
			validURIs = append(validURIs, uri)
		}
	}
	
	if len(validURIs) == 0 {
		fmt.Println("订阅内容中未找到有效的节点链接")
		waitForKeyPress()
		return
	}
	
	// 导入节点
	importNodesFromURIs(validURIs)
}

// 从Base64编码字符串导入
func importFromBase64() {
	clearScreen()
	fmt.Println("===== 从Base64编码字符串导入 =====")
	
	// 获取Base64编码字符串
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("请输入Base64编码字符串: ")
	base64Str, _ := reader.ReadString('\n')
	base64Str = strings.TrimSpace(base64Str)
	
	if base64Str == "" {
		fmt.Println("Base64编码字符串不能为空")
		waitForKeyPress()
		return
	}
	
	// 解码Base64内容
	decodedBytes, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		fmt.Printf("解码Base64内容失败: %v\n", err)
		waitForKeyPress()
		return
	}
	
	// 处理URI列表
	uriList := strings.Split(string(decodedBytes), "\n")
	var validURIs []string
	for _, uri := range uriList {
		uri = strings.TrimSpace(uri)
		if uri != "" {
			validURIs = append(validURIs, uri)
		}
	}
	
	if len(validURIs) == 0 {
		fmt.Println("解码内容中未找到有效的节点链接")
		waitForKeyPress()
		return
	}
	
	// 导入节点
	importNodesFromURIs(validURIs)
}

// 从节点链接(URI)导入
func importFromNodeURIs() {
	clearScreen()
	fmt.Println("===== 从节点链接(URI)导入 =====")
	fmt.Println("支持的链接格式: ss://, vmess://, trojan://")
	fmt.Println("可以一次输入多个链接，每行一个")
	fmt.Println("输入完成后，按Ctrl+D(Linux/Mac)或Ctrl+Z(Windows)或Ctrl+C结束输入")
	fmt.Println("--------------------------------------")
	
	// 创建一个通道来处理Ctrl+C信号
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	
	// 读取用户输入的多行URI
	reader := bufio.NewReader(os.Stdin)
	var uris []string
	
	// 启动一个goroutine来处理信号
	done := make(chan bool, 1)
	go func() {
		<-c
		fmt.Println("\n结束输入")
		done <- true
	}()
	
	inputLoop:
	for {
		select {
		case <-done:
			break inputLoop
		default:
			fmt.Print("> ")
			inputCh := make(chan string, 1)
			errCh := make(chan error, 1)
			
			go func() {
				input, err := reader.ReadString('\n')
				if err != nil {
					errCh <- err
					return
				}
				inputCh <- input
			}()
			
			select {
			case <-done:
				break inputLoop
			case input := <-inputCh:
				uri := strings.TrimSpace(input)
				if uri != "" {
					uris = append(uris, uri)
				}
			case err := <-errCh:
				if err == io.EOF {
					break inputLoop
				}
				fmt.Printf("读取输入失败: %v\n", err)
				waitForKeyPress()
				return
			}
		}
	}
	
	// 取消信号监听
	signal.Stop(c)
	
	if len(uris) == 0 {
		fmt.Println("\n未输入任何链接")
		waitForKeyPress()
		return
	}
	
	fmt.Printf("\n共读取到 %d 个链接\n", len(uris))
	
	// 导入节点
	importNodesFromURIs(uris)
}

// 从YAML文件导入
func importFromYAML() {
	clearScreen()
	fmt.Println("===== 从YAML文件导入 =====")
	
	// 获取YAML文件路径
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("请输入YAML文件路径: ")
	filePath, _ := reader.ReadString('\n')
	filePath = strings.TrimSpace(filePath)
	
	if filePath == "" {
		fmt.Println("文件路径不能为空")
		waitForKeyPress()
		return
	}
	
	// 读取YAML文件
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("读取文件失败: %v\n", err)
		waitForKeyPress()
		return
	}
	
	// 解析YAML
	var yamlConfig map[string]interface{}
	if err := yaml.Unmarshal(content, &yamlConfig); err != nil {
		fmt.Printf("解析YAML失败: %v\n", err)
		waitForKeyPress()
		return
	}
	
	// 提取代理配置
	proxies, ok := yamlConfig["proxies"].([]interface{})
	if !ok || len(proxies) == 0 {
		fmt.Println("YAML中未找到有效的代理配置")
		waitForKeyPress()
		return
	}
	
	// 读取当前配置
	config, err := readClashConfig()
	if err != nil {
		fmt.Printf("读取当前配置失败: %v\n", err)
		waitForKeyPress()
		return
	}
	
	// 获取当前代理列表
	var currentProxies []interface{}
	if existingProxies, ok := config["proxies"].([]interface{}); ok {
		currentProxies = existingProxies
	}
	
	// 当前代理名称集合，用于检查重复
	currentProxyNames := make(map[string]bool)
	for _, p := range currentProxies {
		if proxy, ok := p.(map[string]interface{}); ok {
			if name, ok := proxy["name"].(string); ok {
				currentProxyNames[name] = true
			}
		}
	}
	
	// 添加新代理
	importedCount := 0
	skippedCount := 0
	
	for _, p := range proxies {
		proxy, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		
		// 获取代理名称
		name, ok := proxy["name"].(string)
		if !ok || name == "" {
			skippedCount++
			continue
		}
		
		// 检查是否已存在
		if currentProxyNames[name] {
			fmt.Printf("跳过已存在的代理: %s\n", name)
			skippedCount++
			continue
		}
		
		// 确保必要的字段都存在
		ensureRequiredFields(proxy)
		
		// 添加到当前代理列表
		currentProxies = append(currentProxies, proxy)
		currentProxyNames[name] = true
		
		// 更新代理组
		updateProxyGroup(config, name)
		
		importedCount++
		fmt.Printf("已导入代理: %s\n", name)
	}
	
	// 更新配置
	config["proxies"] = currentProxies
	
	// 保存配置
	if err := saveClashConfig(config); err != nil {
		fmt.Printf("保存配置失败: %v\n", err)
		waitForKeyPress()
		return
	}
	
	fmt.Printf("\n导入完成: 成功导入 %d 个代理，跳过 %d 个代理\n", importedCount, skippedCount)
	
	// 询问是否重启Clash服务
	fmt.Println("是否需要重启Clash服务来应用更改? [y/n]")
	var restart string
	fmt.Scanln(&restart)
	
	if strings.ToLower(restart) == "y" {
		cmd := exec.Command("systemctl", "restart", "clash")
		if err := cmd.Run(); err != nil {
			fmt.Printf("重启Clash服务失败: %v\n", err)
		} else {
			fmt.Println("Clash服务已重启")
		}
	}
	
	waitForKeyPress()
}

// 从URI列表导入节点
func importNodesFromURIs(uris []string) {
	// 读取当前配置
	config, err := readClashConfig()
	if err != nil {
		fmt.Printf("读取配置失败: %v\n", err)
		waitForKeyPress()
		return
	}
	
	// 获取当前代理列表
	var currentProxies []interface{}
	if existingProxies, ok := config["proxies"].([]interface{}); ok {
		currentProxies = existingProxies
	}
	
	// 当前代理名称集合，用于检查重复
	currentProxyNames := make(map[string]bool)
	for _, p := range currentProxies {
		if proxy, ok := p.(map[string]interface{}); ok {
			if name, ok := proxy["name"].(string); ok {
				currentProxyNames[name] = true
			}
		}
	}
	
	// 处理每个URI
	importedCount := 0
	skippedCount := 0
	errorCount := 0
	
	for i, uri := range uris {
		fmt.Printf("处理节点 %d/%d: ", i+1, len(uris))
		
		var proxyConfig map[string]interface{}
		var err error
		
		// 根据URI类型解析
		if strings.HasPrefix(uri, "ss://") {
			proxyConfig, err = parseShadowsocksURI(uri)
		} else if strings.HasPrefix(uri, "vmess://") {
			proxyConfig, err = parseVmessURI(uri)
		} else if strings.HasPrefix(uri, "trojan://") {
			proxyConfig, err = parseTrojanURI(uri)
		} else {
			fmt.Printf("跳过不支持的协议: %s\n", uri[:10]+"...")
			skippedCount++
			continue
		}
		
		if err != nil {
			fmt.Printf("解析失败: %v\n", err)
			errorCount++
			continue
		}
		
		// 获取代理名称
		name, ok := proxyConfig["name"].(string)
		if !ok || name == "" {
			fmt.Println("跳过没有名称的代理")
			skippedCount++
			continue
		}
		
		// 检查是否已存在
		if currentProxyNames[name] {
			fmt.Printf("跳过已存在的代理: %s\n", name)
			skippedCount++
			continue
		}
		
		// 确保必要的字段都存在
		ensureRequiredFields(proxyConfig)
		
		// 添加到当前代理列表
		currentProxies = append(currentProxies, proxyConfig)
		currentProxyNames[name] = true
		
		// 更新代理组
		updateProxyGroup(config, name)
		
		importedCount++
		fmt.Printf("已导入: %s\n", name)
	}
	
	// 更新配置
	config["proxies"] = currentProxies
	
	// 保存配置
	if err := saveClashConfig(config); err != nil {
		fmt.Printf("保存配置失败: %v\n", err)
		waitForKeyPress()
		return
	}
	
	fmt.Printf("\n导入完成: 成功导入 %d 个代理，跳过 %d 个代理，错误 %d 个\n", 
		importedCount, skippedCount, errorCount)
	
	// 询问是否重启Clash服务
	fmt.Println("是否需要重启Clash服务来应用更改? [y/n]")
	var restart string
	fmt.Scanln(&restart)
	
	if strings.ToLower(restart) == "y" {
		cmd := exec.Command("systemctl", "restart", "clash")
		if err := cmd.Run(); err != nil {
			fmt.Printf("重启Clash服务失败: %v\n", err)
		} else {
			fmt.Println("Clash服务已重启")
		}
	}
	
	waitForKeyPress()
}

// 确保代理配置中包含所有必要的字段
func ensureRequiredFields(proxyConfig map[string]interface{}) {
	// 检查代理类型
	proxyType, _ := proxyConfig["type"].(string)
	
	// 添加 udp 字段
	if _, exists := proxyConfig["udp"]; !exists {
		proxyConfig["udp"] = true
	}
	
	// 根据不同代理类型添加必要字段
	switch proxyType {
	case "vmess":
		// 确保 cipher 字段存在
		if _, exists := proxyConfig["cipher"]; !exists {
			proxyConfig["cipher"] = "auto"
		}
		
		// 确保 alterId 字段存在
		if _, exists := proxyConfig["alterId"]; !exists {
			proxyConfig["alterId"] = "0"
		}
		
		// 确保 network 字段存在
		if _, exists := proxyConfig["network"]; !exists {
			proxyConfig["network"] = "tcp"
		}
		
	case "ss", "shadowsocks":
		// 确保 cipher 字段存在
		if _, exists := proxyConfig["cipher"]; !exists {
			proxyConfig["cipher"] = "aes-256-gcm" // 默认加密方式
		}
		
	case "trojan":
		// 确保 skip-cert-verify 字段存在
		if _, exists := proxyConfig["skip-cert-verify"]; !exists {
			proxyConfig["skip-cert-verify"] = false
		}
	}
} 
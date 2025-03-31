package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// 交互式显示代理状态
func interactiveShowProxyStatus() {
	clearScreen()
	fmt.Println("===== 代理节点状态 =====")

	// 检查Clash服务状态
	isRunning := isClashRunning()
	fmt.Printf("Clash服务状态: %s\n\n", statusString(isRunning))

	if !isRunning {
		fmt.Println("Clash服务未运行，无法获取代理状态")
		fmt.Println("请启动Clash服务后再试")
		waitForKeyPress()
		return
	}

	// 检查API可用性
	apiAvailable := true
	client := &http.Client{Timeout: 3 * time.Second}
	_, err := client.Get("http://127.0.0.1:9090/version")
	if err != nil {
		apiAvailable = false
		fmt.Printf("Clash API状态: %s\n", apiStatusString(apiAvailable))
		fmt.Println("无法连接到Clash API，请检查配置")

		// 检查API配置
		apiConfigErr := checkAPIConfig()
		if apiConfigErr != nil {
			fmt.Printf("API配置检查结果: %v\n", apiConfigErr)
		}

		// 尝试诊断API问题
		fmt.Println("\n正在尝试诊断API连接问题...")
		diagErr := diagnosisClashAPI()
		if diagErr != nil {
			fmt.Printf("诊断结果: %v\n", diagErr)
		}

		waitForKeyPress()
		return
	}

	fmt.Printf("Clash API状态: %s\n\n", apiStatusString(apiAvailable))

	// 获取当前选中的代理
	selInfo, selErr := getSelectedProxy()
	if selErr != nil {
		fmt.Printf("获取当前选中代理信息失败: %v\n", selErr)
		waitForKeyPress()
		return
	}

	fmt.Printf("当前选中的代理组: %s\n", selInfo.GroupName)
	fmt.Printf("当前使用的节点: %s\n\n", selInfo.SelectedProxy)

	// 获取所有代理
	proxiesList, proxyErr := getProxies()
	if proxyErr != nil {
		fmt.Printf("获取代理列表失败: %v\n", proxyErr)
		waitForKeyPress()
		return
	}

	if len(proxiesList) == 0 {
		fmt.Println("没有找到任何代理节点")
		waitForKeyPress()
		return
	}

	// 询问是否需要测试节点延迟
	fmt.Print("是否需要测试节点延迟？这将耗费一些时间 [y/n]: ")
	var testDelay string
	fmt.Scanln(&testDelay)

	if strings.ToLower(testDelay) == "y" {
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

		var delayResults map[string]int
		var delayErr error
		bestURL := ""

		switch speedTestMethod {
		case 1:
			delayResults, delayErr = getProxyDelays(proxiesList, useDirectMode)
		case 2:
			delayResults, delayErr = getProxyDelaysSimple(proxiesList)
		case 3:
			delayResults, delayErr = getProxyDelaysReliable(proxiesList)
		default:
			delayResults, delayErr = getProxyDelaysSimple(proxiesList)
		}

		if delayErr != nil {
			fmt.Printf("\n获取代理延迟信息失败: %v\n", delayErr)
		} else {
			// 打印测试结果
			fmt.Println("\n节点延迟测试结果:")
			fmt.Println("----------------------------")

			// 对代理按名称排序，便于查看
			sort.Strings(proxiesList)
			
			for _, proxy := range proxiesList {
				// var delayStr string
				if delay, ok := delayResults[proxy]; ok {
					if delay > 0 {
						var sourceInfo string
						if bestURL != "" {
							sourceInfo = fmt.Sprintf(" (来源: %s)", bestURL)
						}
						fmt.Printf("节点 %s 延迟: %d ms%s\n", 
							proxy, delay, sourceInfo)
					} else {
						fmt.Printf("节点 %s 无法连接\n", proxy)
					}
				} else {
					fmt.Printf("节点 %s 未测试\n", proxy)
				}
			}
			fmt.Println("----------------------------")
		}
	}

	waitForKeyPress()
}

// 切换到直连模式
func switchToDirectMode() error {
	// 获取代理组信息
	resp, err := http.Get("http://127.0.0.1:9090/proxies")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	
	// 解析JSON
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	
	// 获取所有代理信息
	proxies, ok := result["proxies"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("无法获取代理信息")
	}
	
	// 遍历所有代理组，查找Selector类型的组
	for groupName, groupInfo := range proxies {
		group, ok := groupInfo.(map[string]interface{})
		if !ok {
			continue
		}
		
		// 只处理Selector类型的组
		if groupType, ok := group["type"].(string); ok && groupType == "Selector" {
			// 检查是否有DIRECT选项
			all, ok := group["all"].([]interface{})
			if !ok {
				continue
			}
			
			hasDirect := false
			for _, item := range all {
				if name, ok := item.(string); ok && name == "DIRECT" {
					hasDirect = true
					break
				}
			}
			
			if hasDirect {
				// 切换到DIRECT
				if err := switchProxy(groupName, "DIRECT"); err != nil {
					fmt.Printf("切换组 %s 到直连模式失败: %v\n", groupName, err)
				} else {
					fmt.Printf("已将组 %s 切换到直连模式\n", groupName)
				}
			}
		}
	}
	
	return nil
}

// 使用Clash API获取代理延迟
func getProxyDelays(proxyNames []string, useDirectMode bool) (map[string]int, error) {
	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	// 如果使用直连模式，先保存当前代理
	var originalProxy *SelectedProxyInfo
	var err error
	
	if useDirectMode {
		// 获取当前选中的代理
		originalProxy, err = getSelectedProxy()
		if err != nil {
			fmt.Println("警告: 无法获取当前代理设置，将不使用直连模式")
			useDirectMode = false
		} else {
			// 切换到直连模式
			fmt.Println("临时切换到直连模式进行测速...")
			if err := switchToDirectMode(); err != nil {
				fmt.Printf("警告: 切换到直连模式失败: %v，将使用当前代理测速\n", err)
				useDirectMode = false
			}
			// 给一些时间让网络切换生效
			time.Sleep(2 * time.Second)
		}
	}
	
	// 延迟函数：在函数结束时恢复原始代理
	defer func() {
		if useDirectMode && originalProxy != nil {
			fmt.Printf("恢复原始代理设置 (%s -> %s)...\n", 
				originalProxy.GroupName, originalProxy.SelectedProxy)
			
			if err := switchProxy(originalProxy.GroupName, originalProxy.SelectedProxy); err != nil {
				fmt.Printf("恢复原始代理失败: %v\n", err)
			} else {
				fmt.Println("已恢复原始代理设置")
			}
		}
	}()
	
	// 创建延迟结果映射
	delays := make(map[string]int)
	
	// 测试URL列表
	urls := []string{
		"http://www.gstatic.com/generate_204",
		"http://cp.cloudflare.com/generate_204",
		"http://www.qualcomm.cn/generate_204",
	}
	
	// 对每个代理进行测试
	for i, proxyName := range proxyNames {
		fmt.Printf("测试代理 %d/%d: %s\n", i+1, len(proxyNames), proxyName)
		
		// 构造API请求URL
		apiURL := fmt.Sprintf("http://127.0.0.1:9090/proxies/%s/delay", url.PathEscape(proxyName))
		
		// 尝试不同的测速URL，选择可用的
		var bestDelay int = -1
		
		for _, testURL := range urls {
			// 构造查询参数
			reqURL := fmt.Sprintf("%s?url=%s&timeout=5000", apiURL, url.QueryEscape(testURL))
			
			// 发送请求
			req, err := http.NewRequest("GET", reqURL, nil)
			if err != nil {
				fmt.Printf("创建请求失败: %v\n", err)
				continue
			}
			
			resp, err := client.Do(req)
			if err != nil {
				fmt.Printf("请求失败: %v\n", err)
				continue
			}
			
			// 读取响应
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				fmt.Printf("读取响应失败: %v\n", err)
				continue
			}
			
			// 解析响应
			var result struct {
				Delay int `json:"delay"`
			}
			
			if err := json.Unmarshal(body, &result); err != nil {
				fmt.Printf("解析响应失败: %v\n", err)
				continue
			}
			
			// 检查延迟值
			if result.Delay > 0 {
				if bestDelay == -1 || result.Delay < bestDelay {
					bestDelay = result.Delay
				}
			}
		}
		
		// 保存最佳延迟结果
		delays[proxyName] = bestDelay
		
		// 简单的休眠，避免过多请求
		if i < len(proxyNames)-1 {
			time.Sleep(200 * time.Millisecond)
		}
	}
	
	return delays, nil
}

// 使用简化方法进行测速
func getProxyDelaysSimple(proxyNames []string) (map[string]int, error) {
	delays := make(map[string]int)
	
	// 读取配置文件
	config, err := readClashConfig()
	if err != nil {
		return nil, fmt.Errorf("读取配置失败: %v", err)
	}
	
	// 获取代理配置
	proxies, ok := config["proxies"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("配置中未找到代理列表")
	}
	
	fmt.Println("\n使用简化方法测试节点连接...")
	
	for i, name := range proxyNames {
		// 查找该代理的配置
		var proxyConfig map[string]interface{}
		for _, p := range proxies {
			if proxy, ok := p.(map[string]interface{}); ok {
				if proxyName, ok := proxy["name"].(string); ok && proxyName == name {
					proxyConfig = proxy
					break
				}
			}
		}
		
		if proxyConfig == nil {
			fmt.Printf("无法找到代理 %s 的配置\n", name)
			delays[name] = -1
			continue
		}
		
		fmt.Printf("测试代理 %d/%d: %s\n", i+1, len(proxyNames), name)
		
		// 获取服务器和端口
		server, ok := proxyConfig["server"].(string)
		if !ok {
			fmt.Printf("无法获取服务器地址\n")
			delays[name] = -1
			continue
		}
		
		var port string
		if portFloat, ok := proxyConfig["port"].(float64); ok {
			port = fmt.Sprintf("%d", int(portFloat))
		} else if portStr, ok := proxyConfig["port"].(string); ok {
			port = portStr
		} else {
			fmt.Printf("无法获取端口\n")
			delays[name] = -1
			continue
		}
		
		// 创建地址
		address := fmt.Sprintf("%s:%s", server, port)
		
		// 进行连接测试
		var totalDelay time.Duration
		successCount := 0
		testCount := 3
		
		for j := 0; j < testCount; j++ {
			start := time.Now()
			conn, err := net.DialTimeout("tcp", address, 3*time.Second)
			if err != nil {
				continue
			}
			
			elapsed := time.Since(start)
			conn.Close()
			
			totalDelay += elapsed
			successCount++
		}
		
		// 计算平均延迟
		if successCount > 0 {
			avgDelay := int(totalDelay.Milliseconds() / int64(successCount))
			delays[name] = avgDelay
			fmt.Printf("  平均延迟: %d ms\n", avgDelay)
		} else {
			delays[name] = -1
			fmt.Printf("  连接失败\n")
		}
		
		// 简单的休眠，避免过多请求
		if i < len(proxyNames)-1 {
			time.Sleep(200 * time.Millisecond)
		}
	}
	
	return delays, nil
}

// 使用更简单可靠的测速方法
func getProxyDelaysReliable(proxyNames []string) (map[string]int, error) {
	delays := make(map[string]int)
	
	// 读取配置获取代理信息
	config, err := readClashConfig()
	if err != nil {
		return nil, fmt.Errorf("读取配置失败: %v", err)
	}
	
	fmt.Println("\n使用可靠方法测试节点连接...")
	
	for i, name := range proxyNames {
		fmt.Printf("测试节点 %d/%d: %s\n", i+1, len(proxyNames), name)
		
		// 获取服务器 IP
		ip, err := getProxyServerIP(name, config)
		if err != nil {
			fmt.Printf("  无法获取节点IP: %v\n", err)
			delays[name] = -1
			continue
		}
		
		fmt.Printf("  服务器 IP: %s\n", ip)
		
		// 测试方法 1: ICMP Ping (如果系统支持)
		if runtime.GOOS != "windows" { // 在非Windows系统上尝试ICMP ping
			delay := pingTest(ip)
			if delay > 0 {
				fmt.Printf("  ICMP Ping 延迟: %d ms\n", delay)
				delays[name] = delay
				continue // 如果 ping 成功，跳过其他测试
			} else {
				fmt.Println("  ICMP Ping 失败或不可用")
			}
		}
		
		// 测试方法 2: TCP 连接
		delay := tcpConnectTest(ip)
		if delay > 0 {
			fmt.Printf("  TCP 连接延迟: %d ms\n", delay)
			delays[name] = delay
			continue
		} else {
			fmt.Println("  TCP 连接测试失败")
		}
		
		// 如果所有测试都失败
		delays[name] = -1
		fmt.Println("  所有连接测试均失败")
		
		// 简单的休眠，避免过多请求
		if i < len(proxyNames)-1 {
			time.Sleep(200 * time.Millisecond)
		}
	}
	
	return delays, nil
}

// 获取代理服务器IP
func getProxyServerIP(proxyName string, config map[string]interface{}) (string, error) {
	proxies, ok := config["proxies"].([]interface{})
	if !ok {
		return "", fmt.Errorf("配置中未找到代理列表")
	}
	
	// 查找对应的代理
	for _, proxyInterface := range proxies {
		proxy, ok := proxyInterface.(map[string]interface{})
		if !ok {
			continue
		}
		
		name, ok := proxy["name"].(string)
		if !ok || name != proxyName {
			continue
		}
		
		server, ok := proxy["server"].(string)
		if !ok {
			return "", fmt.Errorf("代理配置中未找到服务器信息")
		}
		
		// 尝试解析域名获取IP
		ips, err := net.LookupIP(server)
		if err != nil {
			return "", fmt.Errorf("解析服务器域名失败: %v", err)
		}
		
		if len(ips) == 0 {
			return "", fmt.Errorf("未找到服务器IP")
		}
		
		// 优先返回IPv4地址
		for _, ip := range ips {
			if ipv4 := ip.To4(); ipv4 != nil {
				return ipv4.String(), nil
			}
		}
		
		// 如果没有IPv4地址，返回第一个IP地址
		return ips[0].String(), nil
	}
	
	return "", fmt.Errorf("未找到代理 %s", proxyName)
}

// TCP连接测试
func tcpConnectTest(ip string) int {
	// 测试多个常用端口
	ports := []string{"80", "443", "8080", "1080"}
	
	var bestDelay int64 = -1
	
	for _, port := range ports {
		// 多次测试取平均值
		var totalDelay int64
		successCount := 0
		maxTests := 3
		
		for test := 0; test < maxTests; test++ {
			start := time.Now()
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", ip, port), 3*time.Second)
			if err != nil {
				continue
			}
			elapsed := time.Since(start)
			conn.Close()
			
			totalDelay += elapsed.Milliseconds()
			successCount++
		}
		
		if successCount > 0 {
			avgDelay := totalDelay / int64(successCount)
			if bestDelay == -1 || avgDelay < bestDelay {
				bestDelay = avgDelay
			}
		}
	}
	
	if bestDelay > 0 {
		return int(bestDelay)
	}
	
	return -1
}

// ICMP Ping测试
func pingTest(ip string) int {
	// 创建ping命令，不同系统命令格式可能不同
	var cmd *exec.Cmd
	
	switch runtime.GOOS {
	case "darwin", "linux":
		// macOS和Linux使用 -c 参数指定ping次数
		cmd = exec.Command("ping", "-c", "3", "-W", "1", ip)
	default:
		// Windows或其他系统使用默认参数
		return -1 // 暂不支持Windows的ping测试
	}
	
	// 执行命令并获取输出
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("  Ping失败: %v\n", err)
		return -1
	}
	
	// 解析结果，提取平均延迟
	outputStr := string(output)
	
	// 不同系统输出格式不同，尝试识别常见格式
	var delay int
	
	// 尝试匹配Linux/macOS格式
	if strings.Contains(outputStr, "min/avg/max") {
		// 在Linux/macOS中，格式通常是 "min/avg/max/mdev = 7.851/8.235/8.824/0.414 ms"
		parts := strings.Split(outputStr, "min/avg/max")
		if len(parts) > 1 {
			statsStr := parts[1]
			// 提取平均值
			avgIndex := strings.Index(statsStr, "/")
			if avgIndex != -1 && avgIndex+1 < len(statsStr) {
				avgEndIndex := strings.Index(statsStr[avgIndex+1:], "/")
				if avgEndIndex != -1 {
					avgStr := statsStr[avgIndex+1 : avgIndex+1+avgEndIndex]
					if avgFloat, err := strconv.ParseFloat(avgStr, 64); err == nil {
						delay = int(avgFloat)
					}
				}
			}
		}
	}
	
	if delay > 0 {
		return delay
	}
	
	// 如果无法解析，返回默认值
	return -1
} 
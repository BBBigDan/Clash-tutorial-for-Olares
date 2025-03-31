package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// 切换到新的代理的实现函数
func switchProxy(groupName, proxyName string) error {
	url := fmt.Sprintf("http://127.0.0.1:9090/proxies/%s", groupName)
	
	// 准备请求数据
	requestData := fmt.Sprintf(`{"name":"%s"}`, proxyName)
	
	// 创建请求
	req, err := http.NewRequest("PUT", url, strings.NewReader(requestData))
	if err != nil {
		return err
	}
	
	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	
	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	// 检查响应
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("切换代理失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

// 检查Clash服务是否正在运行的实现函数
func isClashRunning() bool {
	cmd := exec.Command("systemctl", "is-active", "--quiet", "clash")
	err := cmd.Run()
	return err == nil
}

// 获取当前选中的代理
func getSelectedProxy() (*SelectedProxyInfo, error) {
	// 发送请求获取代理组信息
	resp, err := http.Get("http://127.0.0.1:9090/proxies")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	// 解析JSON
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	
	// 获取所有代理信息
	proxies, ok := result["proxies"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("无法获取代理信息")
	}
	
	// 查找选择器类型的代理组
	for groupName, groupInfo := range proxies {
		groupInfoMap, ok := groupInfo.(map[string]interface{})
		if !ok {
			continue
		}
		
		// 只处理类型为Selector的代理组
		if groupType, ok := groupInfoMap["type"].(string); ok && groupType == "Selector" {
			// 尝试获取当前选中的代理
			if selected, ok := groupInfoMap["now"].(string); ok {
				// 跳过包含DIRECT的组
				if selected == "DIRECT" {
					continue
				}
				
				// 找到第一个有效的代理组即返回
				return &SelectedProxyInfo{
					GroupName:     groupName,
					SelectedProxy: selected,
				}, nil
			}
		}
	}
	
	// 如果没有找到任何代理组或选中的代理，返回错误
	return nil, fmt.Errorf("未找到选中的代理")
}

// 获取所有代理列表
func getProxies() ([]string, error) {
	// 发送请求获取代理组信息
	resp, err := http.Get("http://127.0.0.1:9090/proxies")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	// 解析JSON
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	
	// 获取代理组信息
	proxies, ok := result["proxies"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("无法获取代理信息")
	}
	
	// 查找GLOBAL组或第一个Selector类型的组
	var groupInfo map[string]interface{}
	
	// 首先检查是否有GLOBAL组
	if globalGroup, ok := proxies["GLOBAL"].(map[string]interface{}); ok {
		if globalType, ok := globalGroup["type"].(string); ok && globalType == "Selector" {
			groupInfo = globalGroup
		}
	}
	
	// 如果没有找到GLOBAL组，查找第一个Selector类型的组
	if groupInfo == nil {
		for _, value := range proxies {
			if group, ok := value.(map[string]interface{}); ok {
				if groupType, ok := group["type"].(string); ok && groupType == "Selector" {
					groupInfo = group
					break
				}
			}
		}
	}
	
	if groupInfo == nil {
		return nil, fmt.Errorf("未找到可用的代理组")
	}
	
	// 获取所有代理
	all, ok := groupInfo["all"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("无法获取代理列表")
	}
	
	// 转换为字符串数组
	var proxyList []string
	for _, proxy := range all {
		if proxyName, ok := proxy.(string); ok {
			// 排除特殊代理
			if proxyName != "DIRECT" && proxyName != "REJECT" && proxyName != "GLOBAL" {
				proxyList = append(proxyList, proxyName)
			}
		}
	}
	
	return proxyList, nil
}

// 更新代理组的实现函数
func updateProxyGroup(config map[string]interface{}, proxyName string) {
	// 获取代理组配置
	proxyGroups, ok := config["proxy-groups"].([]interface{})
	if !ok {
		return
	}
	
	// 遍历所有代理组，添加新代理到每个组中
	for i, groupInterface := range proxyGroups {
		group, ok := groupInterface.(map[string]interface{})
		if !ok {
			continue
		}
		
		// 获取代理列表
		proxies, ok := group["proxies"].([]interface{})
		if !ok {
			continue
		}
		
		// 检查代理是否已存在
		found := false
		for _, p := range proxies {
			if pName, ok := p.(string); ok && pName == proxyName {
				found = true
				break
			}
		}
		
		// 如果不存在，添加新代理
		if !found {
			proxies = append(proxies, proxyName)
			group["proxies"] = proxies
			proxyGroups[i] = group
		}
	}
	
	// 更新配置
	config["proxy-groups"] = proxyGroups
}

// 从代理组中移除代理的实现函数
func removeFromProxyGroups(config map[string]interface{}, proxyName string) {
	// 获取代理组配置
	proxyGroups, ok := config["proxy-groups"].([]interface{})
	if !ok {
		return
	}
	
	// 遍历所有代理组，移除指定代理
	for i, groupInterface := range proxyGroups {
		group, ok := groupInterface.(map[string]interface{})
		if !ok {
			continue
		}
		
		// 获取代理列表
		proxiesInterface, ok := group["proxies"].([]interface{})
		if !ok {
			continue
		}
		
		// 创建新的代理列表，排除要删除的代理
		var newProxies []interface{}
		for _, p := range proxiesInterface {
			if pName, ok := p.(string); ok && pName != proxyName {
				newProxies = append(newProxies, p)
			}
		}
		
		// 更新代理列表
		group["proxies"] = newProxies
		proxyGroups[i] = group
		
		// 如果被删除的代理是当前选中的代理，更新selected字段
		if selected, ok := group["selected"].(string); ok && selected == proxyName {
			if len(newProxies) > 0 {
				if firstProxy, ok := newProxies[0].(string); ok {
					group["selected"] = firstProxy
				}
			} else {
				delete(group, "selected")
			}
			proxyGroups[i] = group
		}
	}
	
	// 更新配置
	config["proxy-groups"] = proxyGroups
}

// 检查配置文件是否正确启用了API
func checkAPIConfig() error {
	config, err := readClashConfig()
	if err != nil {
		return fmt.Errorf("读取配置失败: %v", err)
	}
	
	// 检查外部控制设置
	externalController, ok := config["external-controller"].(string)
	if !ok || externalController == "" {
		return fmt.Errorf("配置文件中未找到 external-controller 设置或设置为空")
	}
	
	// 检查端口是否为9090
	if !strings.Contains(externalController, "9090") {
		return fmt.Errorf("external-controller 端口不是9090，当前设置为: %s", externalController)
	}
	
	// 检查是否允许外部访问
	if strings.HasPrefix(externalController, "127.0.0.1") || strings.HasPrefix(externalController, "localhost") {
		fmt.Println("提示: API仅允许本地访问")
	} else if strings.HasPrefix(externalController, "0.0.0.0") {
		fmt.Println("提示: API允许所有网络接口访问")
	}
	
	// 检查API密钥
	if secret, ok := config["secret"].(string); ok && secret != "" {
		fmt.Println("提示: API已设置访问密钥")
	} else {
		fmt.Println("提示: API未设置访问密钥，可能存在安全风险")
	}
	
	// 检查UI设置
	if externalUI, ok := config["external-ui"].(string); ok && externalUI != "" {
		fmt.Printf("提示: 已配置Web UI，路径为: %s\n", externalUI)
	} else {
		fmt.Println("提示: 未配置Web UI")
	}
	
	return nil
}

// 添加一个诊断函数，检查Clash API可用性
func diagnosisClashAPI() error {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	
	endpoints := []string{
		"http://127.0.0.1:9090/version",
		"http://127.0.0.1:9090/configs",
		"http://127.0.0.1:9090/proxies",
		"http://127.0.0.1:9090/rules",
	}
	
	fmt.Println("正在检查Clash API端点:")
	
	for _, endpoint := range endpoints {
		fmt.Printf("测试端点: %s...", endpoint)
		resp, err := client.Get(endpoint)
		if err != nil {
			fmt.Printf("失败: %v\n", err)
			return fmt.Errorf("无法连接到API端点 %s: %v", endpoint, err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			fmt.Printf("失败: 状态码 %d\n", resp.StatusCode)
			fmt.Printf("响应内容: %s\n", string(body))
			return fmt.Errorf("API端点 %s 返回非正常状态码: %d", endpoint, resp.StatusCode)
		}
		
		fmt.Println("成功")
	}
	
	fmt.Println("API检查完成，所有端点都可以访问")
	return nil
}

// 辅助函数：状态字符串
func statusString(isRunning bool) string {
	if isRunning {
		return "运行中"
	}
	return "已停止"
}

// 辅助函数：API状态字符串
func apiStatusString(isAvailable bool) string {
	if isAvailable {
		return "可用"
	}
	return "不可用"
} 
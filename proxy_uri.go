package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// 解析Shadowsocks URI格式的实现函数
func parseShadowsocksURI(uri string) (map[string]interface{}, error) {
	// 移除协议前缀
	if !strings.HasPrefix(uri, "ss://") {
		return nil, fmt.Errorf("不是有效的Shadowsocks URI")
	}
	
	// 分离基本部分和Fragment部分（名称）
	uri = uri[5:] // 去掉"ss://"
	var name string
	if idx := strings.Index(uri, "#"); idx != -1 {
		name = uri[idx+1:]
		name, _ = url.QueryUnescape(name) // 解码URL编码的名称
		uri = uri[:idx]
	}
	
	// 处理两种格式的Shadowsocks URI
	// 1. ss://BASE64(method:password@host:port)
	// 2. ss://BASE64(method:password)@host:port
	
	// 尝试解析第二种格式 ss://BASE64(method:password)@host:port
	parts := strings.SplitN(uri, "@", 2)
	if len(parts) == 2 {
		// 解析方法和密码
		methodAndPass, err := base64.StdEncoding.DecodeString(parts[0])
		if err != nil {
			// 尝试下一种格式
			goto parseFormat1
		}
		
		methodAndPassStr := string(methodAndPass)
		mpParts := strings.SplitN(methodAndPassStr, ":", 2)
		if len(mpParts) != 2 {
			return nil, fmt.Errorf("无效的方法和密码格式")
		}
		
		method := mpParts[0]
		password := mpParts[1]
		
		// 解析主机和端口
		hostAndPort := parts[1]
		hpParts := strings.SplitN(hostAndPort, ":", 2)
		if len(hpParts) != 2 {
			return nil, fmt.Errorf("无效的主机和端口格式")
		}
		
		host := hpParts[0]
		port := hpParts[1]
		
		// 创建代理配置
		proxyMap := make(map[string]interface{})
		proxyMap["type"] = "ss"
		proxyMap["server"] = host
		proxyMap["port"] = port
		proxyMap["cipher"] = method
		proxyMap["password"] = password
		
		// 设置名称
		if name == "" {
			name = fmt.Sprintf("SS-%s:%s", host, port)
		}
		proxyMap["name"] = name
		
		// 附加选项
		proxyMap["udp"] = true
		
		return proxyMap, nil
	}
	
parseFormat1:
	// 尝试解析第一种格式 ss://BASE64(method:password@host:port)
	decoded, err := base64.StdEncoding.DecodeString(uri)
	if err != nil {
		return nil, fmt.Errorf("解码Base64失败: %v", err)
	}
	
	decodedStr := string(decoded)
	userInfoParts := strings.SplitN(decodedStr, "@", 2)
	if len(userInfoParts) != 2 {
		return nil, fmt.Errorf("无效的URI格式")
	}
	
	methodAndPass := userInfoParts[0]
	mpParts := strings.SplitN(methodAndPass, ":", 2)
	if len(mpParts) != 2 {
		return nil, fmt.Errorf("无效的方法和密码格式")
	}
	
	method := mpParts[0]
	password := mpParts[1]
	
	hostAndPort := userInfoParts[1]
	hpParts := strings.SplitN(hostAndPort, ":", 2)
	if len(hpParts) != 2 {
		return nil, fmt.Errorf("无效的主机和端口格式")
	}
	
	host := hpParts[0]
	port := hpParts[1]
	
	// 创建代理配置
	proxyMap := make(map[string]interface{})
	proxyMap["type"] = "ss"
	proxyMap["server"] = host
	proxyMap["port"] = port
	proxyMap["cipher"] = method
	proxyMap["password"] = password
	
	// 设置名称
	if name == "" {
		name = fmt.Sprintf("SS-%s:%s", host, port)
	}
	proxyMap["name"] = name
	
	// 附加选项
	proxyMap["udp"] = true
	
	return proxyMap, nil
}

// 解析VMess URI格式的实现函数
func parseVmessURI(uri string) (map[string]interface{}, error) {
	// 移除协议前缀
	if !strings.HasPrefix(uri, "vmess://") {
		return nil, fmt.Errorf("不是有效的VMess URI")
	}
	
	// 解码Base64部分
	base64Str := uri[8:] // 去掉"vmess://"
	decoded, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return nil, fmt.Errorf("解码Base64失败: %v", err)
	}
	
	// 解析JSON
	var vmessConfig map[string]interface{}
	if err := json.Unmarshal(decoded, &vmessConfig); err != nil {
		// 尝试替代格式
		return nil, fmt.Errorf("解析VMess配置失败: %v", err)
	}
	
	// 提取配置信息
	proxyMap := make(map[string]interface{})
	proxyMap["type"] = "vmess"
	
	// 必要字段
	addr, ok := vmessConfig["add"].(string)
	if !ok {
		return nil, fmt.Errorf("VMess配置缺少地址字段")
	}
	proxyMap["server"] = addr
	
	port, ok := vmessConfig["port"]
	if !ok {
		return nil, fmt.Errorf("VMess配置缺少端口字段")
	}
	
	// 处理端口类型（可能是字符串或数字）
	switch p := port.(type) {
	case float64:
		proxyMap["port"] = strconv.FormatInt(int64(p), 10)
	case string:
		proxyMap["port"] = p
	default:
		proxyMap["port"] = fmt.Sprintf("%v", port)
	}
	
	id, ok := vmessConfig["id"].(string)
	if !ok {
		return nil, fmt.Errorf("VMess配置缺少ID字段")
	}
	proxyMap["uuid"] = id
	
	// 可选字段
	if aid, ok := vmessConfig["aid"]; ok {
		switch a := aid.(type) {
		case float64:
			proxyMap["alterId"] = strconv.FormatInt(int64(a), 10)
		case string:
			proxyMap["alterId"] = a
		default:
			proxyMap["alterId"] = fmt.Sprintf("%v", aid)
		}
	} else {
		proxyMap["alterId"] = "0"
	}

	if net, ok := vmessConfig["net"].(string); ok {
		proxyMap["network"] = net
	} else {
		proxyMap["network"] = "tcp"
	}
	
	if tls, ok := vmessConfig["tls"].(string); ok && tls == "tls" {
		proxyMap["tls"] = true
	}
	
	if host, ok := vmessConfig["host"].(string); ok {
		proxyMap["ws-headers"] = map[string]interface{}{
			"Host": host,
		}
		
		if sni, ok := vmessConfig["sni"].(string); ok && sni != "" {
			proxyMap["servername"] = sni
		} else {
			proxyMap["servername"] = host
		}
	}
	
	if path, ok := vmessConfig["path"].(string); ok {
		proxyMap["ws-path"] = path
	}
	
	// 设置名称
	var name string
	if ps, ok := vmessConfig["ps"].(string); ok && ps != "" {
		name = ps
	} else {
		name = fmt.Sprintf("VMess-%s:%s", proxyMap["server"], proxyMap["port"])
	}
	proxyMap["name"] = name
	
	// 附加选项
	proxyMap["udp"] = true
	
	return proxyMap, nil
}

// 解析Trojan URI格式的实现函数
func parseTrojanURI(uri string) (map[string]interface{}, error) {
	if !strings.HasPrefix(uri, "trojan://") {
		return nil, fmt.Errorf("不是有效的Trojan URI")
	}
	
	// 移除协议前缀
	uri = uri[9:] // 去掉"trojan://"
	
	// 解析名称部分（在#后面）
	var name string
	if idx := strings.Index(uri, "#"); idx != -1 {
		name = uri[idx+1:]
		name, _ = url.QueryUnescape(name) // 解码URL编码的名称
		uri = uri[:idx]
	}
	
	// 解析查询参数
	var queryStr string
	if idx := strings.Index(uri, "?"); idx != -1 {
		queryStr = uri[idx+1:]
		uri = uri[:idx]
	}
	
	// 解析主要部分（密码@服务器:端口）
	parts := strings.SplitN(uri, "@", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("无效的Trojan URI格式，缺少@分隔符")
	}
	
	password := parts[0]
	serverPort := parts[1]
	
	// 解析服务器和端口
	serverParts := strings.Split(serverPort, ":")
	if len(serverParts) != 2 {
		return nil, fmt.Errorf("无效的服务器地址和端口格式")
	}
	
	server := serverParts[0]
	port := serverParts[1]
	
	// 创建代理配置
	proxyMap := make(map[string]interface{})
	proxyMap["type"] = "trojan"
	proxyMap["server"] = server
	proxyMap["port"] = port
	proxyMap["password"] = password
	
	// 处理查询参数
	if queryStr != "" {
		values, err := url.ParseQuery(queryStr)
		if err == nil {
			// 处理SNI
			if sni := values.Get("peer"); sni != "" {
				proxyMap["sni"] = sni
			} else if sni := values.Get("sni"); sni != "" {
				proxyMap["sni"] = sni
			}
			
			// 处理allowInsecure
			if allowInsecure := values.Get("allowInsecure"); allowInsecure == "1" {
				proxyMap["skip-cert-verify"] = true
			}
		}
	}
	
	// 设置名称
	if name == "" {
		name = fmt.Sprintf("Trojan-%s:%s", server, port)
	}
	proxyMap["name"] = name
	
	// 附加选项
	proxyMap["udp"] = true
	
	return proxyMap, nil
} 
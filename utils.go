package main

import (
	"fmt"
	"os"
	"os/exec"
)

// VerifyInstallation 检查 Clash 是否正确安装
func VerifyInstallation() error {
	// 检查 clash 二进制文件
	_, err := exec.LookPath("clash")
	if err != nil {
		return fmt.Errorf("Clash 二进制文件未找到: %v", err)
	}
	
	// 检查配置文件
	if _, err := os.Stat("/srv/clash/config.yaml"); os.IsNotExist(err) {
		return fmt.Errorf("Clash 配置文件未找到: %v", err)
	}
	
	// 检查 Country.mmdb 文件
	mmdbPaths := []string{
		"/root/.config/clash/Country.mmdb",
		os.ExpandEnv("$HOME/.config/clash/Country.mmdb"),
	}
	
	mmdbFound := false
	for _, path := range mmdbPaths {
		if _, err := os.Stat(path); err == nil {
			mmdbFound = true
			break
		}
	}
	
	if !mmdbFound {
		return fmt.Errorf("Country.mmdb 文件未找到")
	}
	
	return nil
} 
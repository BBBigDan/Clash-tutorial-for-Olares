package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed resources
var embeddedResources embed.FS

// 提取内嵌资源到临时目录
func extractEmbeddedResources(workDir string) error {
	fmt.Println("正在详细记录嵌入资源提取过程...")
	fmt.Println("开始读取嵌入的资源...")
	
	// 将内嵌的clash-premium-installer解压到工作目录
	installerDir := filepath.Join(workDir, "clash-premium-installer")
	err := os.MkdirAll(installerDir, 0755)
	if err != nil {
		return err
	}
	
	fmt.Println("开始解压资源到临时目录...")
	// 递归解压资源
	err = extractDir("resources/clash-premium-installer", installerDir)
	if err != nil {
		return err
	}
	
	fmt.Println("验证提取的文件...")
	// 验证安装脚本是否存在
	installerScript := filepath.Join(installerDir, "installer.sh")
	if _, err := os.Stat(installerScript); os.IsNotExist(err) {
		return fmt.Errorf("安装脚本未找到: %s", installerScript)
	}
	
	// 确保安装脚本有执行权限
	err = os.Chmod(installerScript, 0755)
	if err != nil {
		return err
	}
	
	return nil
}

// 递归解压目录
func extractDir(embeddedPath string, targetDir string) error {
	entries, err := embeddedResources.ReadDir(embeddedPath)
	if err != nil {
		return err
	}
	
	for _, entry := range entries {
		embeddedEntryPath := filepath.Join(embeddedPath, entry.Name())
		targetPath := filepath.Join(targetDir, entry.Name())
		
		if entry.IsDir() {
			// 创建目录
			err := os.MkdirAll(targetPath, 0755)
			if err != nil {
				return err
			}
			
			// 递归解压子目录
			err = extractDir(embeddedEntryPath, targetPath)
			if err != nil {
				return err
			}
		} else {
			// 解压文件
			err := extractFile(embeddedEntryPath, targetPath)
			if err != nil {
				return err
			}
		}
	}
	
	return nil
}

// 解压单个文件
func extractFile(embeddedPath string, targetPath string) error {
	// 读取嵌入文件
	data, err := embeddedResources.ReadFile(embeddedPath)
	if err != nil {
		return err
	}
	
	// 写入目标文件
	err = os.WriteFile(targetPath, data, 0644)
	if err != nil {
		return err
	}
	
	// 如果是脚本文件，添加执行权限
	if filepath.Ext(targetPath) == ".sh" {
		err = os.Chmod(targetPath, 0755)
		if err != nil {
			return err
		}
	}
	
	return nil
}

// 从嵌入资源中读取文件
func readEmbeddedFile(path string) ([]byte, error) {
	return embeddedResources.ReadFile(path)
} 
package main

import (
	"log"
	"os"

	"pixly/internal/cmd"     // 更新为正确的导入路径
	"pixly/internal/version" // 更新为正确的导入路径
)

func main() {
	// 设置版本信息
	cmd.SetVersionInfo(version.GetVersion(), version.GetBuildTime()) // 更新为正确的函数调用

	if err := cmd.Execute(); err != nil { // 更新为正确的函数调用
		log.Fatalf("Error executing command: %v", err)
		os.Exit(1)
	}
}

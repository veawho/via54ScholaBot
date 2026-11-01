// Command schola is the via54ScholaBot CLI entry point.
//
// [CN] via54ScholaBot 命令行入口：深度研究引擎。
// [EN] Command-line interface entry point for via54ScholaBot deep-research engine.
//
// Usage:
//
//	schola [global flags] <subcommand> [args]
//
// Subcommands:
//
//	research   深度研究（学术/报告/文献全链路）
//	academic   学术检索及研究
//	report     报告检索及研究
//	lit        文献检索及研究
//	memory     三层记忆管理
//	skill      科研技能库
//	doctor     环境自检
//	version    版本信息
package main

import (
	"fmt"
	"os"

	"github.com/veawho/via54ScholaBot/cmd/schola/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

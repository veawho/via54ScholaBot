// Package commands implements the via54ScholaBot CLI subcommands.
package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/veawho/via54ScholaBot/internal/version"
)

var rootCmd = &cobra.Command{
	Use:   "schola",
	Short: "via54ScholaBot - 深度研究引擎 / Deep Research Engine",
	Long: `via54ScholaBot: 集学术检索及研究、报告检索及研究、文献检索及研究于一体的深度研究引擎。

集学术检索及研究、报告检索及研究、文献检索及研究于一体的深度研究引擎。
Unite academic search, report research and literature research into one deep-research engine.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// Execute 运行 CLI
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(
		newResearchCmd(),
		newAcademicCmd(),
		newReportCmd(),
		newLitCmd(),
		newMemoryCmd(),
		newSkillCmd(),
		newDoctorCmd(),
		newVersionCmd(),
	)
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "显示版本信息",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version.Full())
		},
	}
}

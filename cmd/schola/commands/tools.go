package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/veawho/via54ScholaBot/internal/schola"
)

func newMemoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memory",
		Short: "三层记忆管理（会话/项目/长期）",
	}
	cmd.AddCommand(
		newMemorySaveCmd(),
		newMemoryListCmd(),
		newMemoryForgetCmd(),
	)
	return cmd
}

func newMemorySaveCmd() *cobra.Command {
	var level, project string
	cmd := &cobra.Command{
		Use:   "save <标题> <内容>",
		Short: "保存一条记忆",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			s := schola.NewMemoryStore("")
			id, err := s.Save(schola.MemoryEntry{
				Level: level, Project: project, Title: args[0], Content: args[1],
			})
			if err != nil {
				return err
			}
			fmt.Printf("已保存记忆 %s → %s\n", id, s)
			return nil
		},
	}
	cmd.Flags().StringVar(&level, "level", "project", "层级: session|project|longterm")
	cmd.Flags().StringVar(&project, "project", "via54ScholaBot", "所属项目")
	return cmd
}

func newMemoryListCmd() *cobra.Command {
	var level, project string
	var limit int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出记忆",
		Run: func(cmd *cobra.Command, args []string) {
			s := schola.NewMemoryStore("")
			entries := s.List(level, project, limit)
			if len(entries) == 0 {
				fmt.Println("（无匹配记忆）")
				return
			}
			for _, e := range entries {
				fmt.Printf("[%s] %s | %s\n  %s\n", e.Level, e.Title, e.CreatedAt, e.Content)
			}
		},
	}
	cmd.Flags().StringVar(&level, "level", "", "按层级过滤: session|project|longterm")
	cmd.Flags().StringVar(&project, "project", "", "按项目过滤")
	cmd.Flags().IntVar(&limit, "limit", 20, "最大条数")
	return cmd
}

func newMemoryForgetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "forget <ID>",
		Short: "删除一条记忆",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s := schola.NewMemoryStore("")
			ok, err := s.Forget(strings.TrimSpace(args[0]))
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("未找到记忆 %s", args[0])
			}
			fmt.Println("已删除", args[0])
			return nil
		},
	}
	return cmd
}

func newSkillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "科研技能库",
	}
	cmd.AddCommand(
		newSkillListCmd(),
		newSkillRunCmd(),
	)
	return cmd
}

func newSkillListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "列出全部技能",
		Run: func(cmd *cobra.Command, args []string) {
			for _, sk := range schola.NewSkills().List() {
				fmt.Printf("%s — %s (%s)\n", sk.Name, sk.Description, sk.Kind)
			}
		},
	}
}

func newSkillRunCmd() *cobra.Command {
	var sources string
	cmd := &cobra.Command{
		Use:   "run <技能名>",
		Short: "运行技能（需提供证据，可用 --sources 逗号分隔源名）",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s := schola.NewSkills()
			sk, ok := s.Get(args[0])
			if !ok {
				return fmt.Errorf("未知技能 %s", args[0])
			}
			hits := []schola.SourceHit{}
			for _, name := range strings.Split(sources, ",") {
				if name != "" {
					hits = append(hits, schola.SourceHit{Source: name, Title: "示例证据", Year: "2026"})
				}
			}
			out, err := s.Run(sk.Name, hits)
			if err != nil {
				return err
			}
			fmt.Println(out)
			return nil
		},
	}
	cmd.Flags().StringVar(&sources, "sources", "openalex", "逗号分隔的证据来源")
	return cmd
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "环境自检",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("via54ScholaBot 环境自检 / Doctor:")
			s := schola.NewMemoryStore("")
			fmt.Println("  [OK] 记忆库:", s)
			fmt.Printf("  [OK] 技能库: %d 项\n", len(schola.NewSkills().List()))
			fmt.Printf("  [OK] 数据源: %d 项\n", len(schola.NewSourceRegistry(nil)))
			avail := schola.NewModelRouter("").Available()
			if len(avail) == 0 {
				fmt.Println("  [--] 模型路由: 未配置 API Key（BYOK 可稍后配置）")
			} else {
				names := make([]string, 0, len(avail))
				for _, p := range avail {
					names = append(names, p.Name)
				}
				fmt.Printf("  [OK] 模型路由: %s\n", strings.Join(names, ", "))
			}
			fmt.Println("  完成。祝研究顺利。")
		},
	}
}

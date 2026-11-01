package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/veawho/via54ScholaBot/internal/schola"
)

func newResearchCmd() *cobra.Command {
	var scope, lang, out string
	var depth int
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "research <研究问题>",
		Short: "深度研究：规划→检索→综合→报告",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			q := strings.Join(args, " ")
			task := schola.ResearchTask{Question: q, Scope: scope, Depth: depth, Lang: lang}
			if dryRun {
				plan := schola.Plan(task)
				fmt.Println("研究计划 / Research Plan:")
				fmt.Printf("  问题 / Question: %s\n", plan.Question)
				fmt.Printf("  关键词 / Keywords: %s\n", strings.Join(plan.Keywords, ", "))
				fmt.Printf("  数据源 / Sources: %s\n", strings.Join(plan.Sources, ", "))
				for _, s := range plan.Steps {
					fmt.Printf("  %d. %s — %s\n", s.Order, s.Action, s.Detail)
				}
				return nil
			}
			o := schola.NewOrchestrator(nil, nil, nil)
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()
			rep, err := o.Research(ctx, task)
			if err != nil {
				return err
			}
			md := schola.RenderMarkdown(rep)
			if out != "" {
				return os.WriteFile(out, []byte(md), 0o644)
			}
			fmt.Println(md)
			return nil
		},
	}
	cmd.Flags().StringVar(&scope, "scope", "all", "研究范围: academic|report|literature|all")
	cmd.Flags().IntVar(&depth, "depth", 2, "研究深度 1-3")
	cmd.Flags().StringVar(&lang, "lang", "zh", "输出语言: zh|en")
	cmd.Flags().StringVar(&out, "out", "", "输出文件路径（.md）")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "仅打印研究计划，不发起检索")
	return cmd
}

func newAcademicCmd() *cobra.Command {
	var limit int
	var lang, out string
	cmd := &cobra.Command{
		Use:   "academic <检索词>",
		Short: "学术检索及研究（多源并发）",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			q := strings.Join(args, " ")
			task := schola.ResearchTask{Question: q, Scope: "academic", Depth: 1, Lang: lang}
			o := schola.NewOrchestrator(nil, nil, nil)
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()
			rep, err := o.Research(ctx, task)
			if err != nil {
				return err
			}
			md := schola.RenderMarkdown(rep)
			if out != "" {
				return os.WriteFile(out, []byte(md), 0o644)
			}
			fmt.Println(md)
			return nil
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 5, "每源结果数（保留）")
	cmd.Flags().StringVar(&lang, "lang", "zh", "输出语言: zh|en")
	cmd.Flags().StringVar(&out, "out", "", "输出文件路径（.md）")
	return cmd
}

func newReportCmd() *cobra.Command {
	var lang, out string
	cmd := &cobra.Command{
		Use:   "report <检索词>",
		Short: "报告检索及研究（SEC/OpenFDA/Crossref）",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			q := strings.Join(args, " ")
			task := schola.ResearchTask{Question: q, Scope: "report", Depth: 1, Lang: lang}
			o := schola.NewOrchestrator(nil, nil, nil)
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()
			rep, err := o.Research(ctx, task)
			if err != nil {
				return err
			}
			md := schola.RenderMarkdown(rep)
			if out != "" {
				return os.WriteFile(out, []byte(md), 0o644)
			}
			fmt.Println(md)
			return nil
		},
	}
	cmd.Flags().StringVar(&lang, "lang", "zh", "输出语言: zh|en")
	cmd.Flags().StringVar(&out, "out", "", "输出文件路径（.md）")
	return cmd
}

func newLitCmd() *cobra.Command {
	var out string
	cmd := &cobra.Command{
		Use:   "lit <DOI>",
		Short: "文献检索及研究（全文开放获取定位）",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			doi := args[0]
			task := schola.ResearchTask{Question: doi, Scope: "literature", Depth: 1, Lang: "zh"}
			o := schola.NewOrchestrator(nil, nil, nil)
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			rep, err := o.Research(ctx, task)
			if err != nil {
				return err
			}
			md := schola.RenderMarkdown(rep)
			if out != "" {
				return os.WriteFile(out, []byte(md), 0o644)
			}
			fmt.Println(md)
			return nil
		},
	}
	cmd.Flags().StringVar(&out, "out", "", "输出文件路径（.md）")
	return cmd
}

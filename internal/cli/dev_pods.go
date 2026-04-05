package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/dag7/ironplate/internal/devtools"
	"github.com/dag7/ironplate/internal/tui"
	"github.com/spf13/cobra"
)

func newDevPodsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pods",
		Aliases: []string{"pod"},
		Short:   "Pod management and debugging",
	}

	cmd.AddCommand(&cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List pods grouped by service",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, cfg, err := findProject()
			if err != nil {
				return err
			}
			return listPodsGrouped(cfg.Metadata.Name)
		},
	})

	cmd.AddCommand(newDevPodsLogsCmd())
	cmd.AddCommand(newDevPodsExecCmd())
	cmd.AddCommand(newDevPodsPortForwardCmd())
	cmd.AddCommand(newDevPodsDescribeCmd())

	return cmd
}

func newDevPodsLogsCmd() *cobra.Command {
	var (
		follow    bool
		tail      int
		container string
	)
	cmd := &cobra.Command{
		Use:   "logs [pod]",
		Short: "View pod logs",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, cfg, err := findProject()
			if err != nil {
				return err
			}
			podName := ""
			if len(args) > 0 {
				podName = args[0]
			} else {
				podName, err = selectPod(cfg.Metadata.Name)
				if err != nil || podName == "" {
					return err
				}
			}
			return devtools.PodLogs(podName, cfg.Metadata.Name, follow, tail, container)
		},
	}
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	cmd.Flags().IntVarP(&tail, "tail", "t", 100, "Number of lines from the end")
	cmd.Flags().StringVarP(&container, "container", "c", "", "Container name")
	return cmd
}

func newDevPodsExecCmd() *cobra.Command {
	var container string
	cmd := &cobra.Command{
		Use:     "exec [pod]",
		Aliases: []string{"sh"},
		Short:   "Shell into a pod",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, cfg, err := findProject()
			if err != nil {
				return err
			}
			podName := ""
			if len(args) > 0 {
				podName = args[0]
			} else {
				podName, err = selectPod(cfg.Metadata.Name)
				if err != nil || podName == "" {
					return err
				}
			}
			return devtools.PodExec(podName, cfg.Metadata.Name, container)
		},
	}
	cmd.Flags().StringVarP(&container, "container", "c", "", "Container name")
	return cmd
}

func newDevPodsPortForwardCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "port-forward [pod] [local:remote]",
		Aliases: []string{"pf"},
		Short:   "Forward ports to a pod",
		Args:    cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, cfg, err := findProject()
			if err != nil {
				return err
			}

			podName := ""
			if len(args) >= 1 {
				podName = args[0]
			} else {
				podName, err = selectPod(cfg.Metadata.Name)
				if err != nil || podName == "" {
					return err
				}
			}

			var local, remote int
			if len(args) >= 2 {
				if _, err := fmt.Sscanf(args[1], "%d:%d", &local, &remote); err != nil {
					return fmt.Errorf("invalid port format, use local:remote (e.g. 8080:80)")
				}
			} else {
				// Interactive port input
				var ports string
				if err := huh.NewInput().
					Title("Ports (local:remote, e.g. 8080:80)").
					Value(&ports).
					Run(); err != nil {
					return nil
				}
				if _, err := fmt.Sscanf(ports, "%d:%d", &local, &remote); err != nil {
					return fmt.Errorf("invalid port format, use local:remote (e.g. 8080:80)")
				}
			}

			return devtools.PodPortForward(podName, cfg.Metadata.Name, local, remote)
		},
	}
}

func newDevPodsDescribeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "describe [pod]",
		Short: "Describe a pod",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, cfg, err := findProject()
			if err != nil {
				return err
			}
			podName := ""
			if len(args) > 0 {
				podName = args[0]
			} else {
				podName, err = selectPod(cfg.Metadata.Name)
				if err != nil || podName == "" {
					return err
				}
			}
			return devtools.DescribePod(podName, cfg.Metadata.Name)
		},
	}
}

func runPodsInteractive(namespace string) error {
	var choice string
	if err := huh.NewSelect[string]().
		Title("Pods").
		Options(
			huh.NewOption("List pods", "list"),
			huh.NewOption("View logs", "logs"),
			huh.NewOption("Shell into pod", "exec"),
			huh.NewOption("Port forward", "port-forward"),
			huh.NewOption("Describe pod", "describe"),
			huh.NewOption("Back", "back"),
		).
		Value(&choice).
		Run(); err != nil {
		return nil
	}

	switch choice {
	case "list":
		return listPodsGrouped(namespace)
	case "logs":
		pod, err := selectPod(namespace)
		if err != nil || pod == "" {
			return err
		}
		// Tail lines selection
		var tailLines int
		if err := huh.NewSelect[int]().
			Title("Tail lines").
			Options(
				huh.NewOption("50 lines", 50),
				huh.NewOption("100 lines", 100),
				huh.NewOption("500 lines", 500),
				huh.NewOption("1000 lines", 1000),
			).
			Value(&tailLines).
			Run(); err != nil {
			tailLines = 100
		}
		return devtools.PodLogs(pod, namespace, true, tailLines, "")
	case "exec":
		pod, err := selectPod(namespace)
		if err != nil || pod == "" {
			return err
		}
		return devtools.PodExec(pod, namespace, "")
	case "port-forward":
		pod, err := selectPod(namespace)
		if err != nil || pod == "" {
			return err
		}
		var ports string
		if err := huh.NewInput().
			Title("Ports (local:remote, e.g. 8080:80)").
			Value(&ports).
			Run(); err != nil {
			return nil
		}
		var local, remote int
		if _, err := fmt.Sscanf(ports, "%d:%d", &local, &remote); err != nil {
			return fmt.Errorf("invalid port format, use local:remote")
		}
		return devtools.PodPortForward(pod, namespace, local, remote)
	case "describe":
		pod, err := selectPod(namespace)
		if err != nil || pod == "" {
			return err
		}
		return devtools.DescribePod(pod, namespace)
	}
	return nil
}

func listPodsGrouped(namespace string) error {
	pods, err := devtools.ListPods(namespace)
	if err != nil {
		return err
	}

	if len(pods) == 0 {
		tui.NewStatusPrinter().Info("No pods found in namespace " + namespace)
		return nil
	}

	groups := devtools.GroupPodsByService(pods)

	fmt.Println()
	for _, g := range groups {
		fmt.Printf("  %s\n", tui.BoldStyle.Render(g.Name))
		for _, p := range g.Pods {
			statusStyle := tui.MutedStyle
			switch strings.ToLower(p.Status) {
			case "running":
				statusStyle = tui.SuccessStyle
			case "pending":
				statusStyle = tui.WarningStyle
			case "failed", "error", "crashloopbackoff":
				statusStyle = tui.ErrorStyle
			}

			restartInfo := ""
			if p.Restarts > 0 {
				restartInfo = tui.WarningStyle.Render(fmt.Sprintf(" (%d restarts)", p.Restarts))
			}

			readyInfo := ""
			if p.Ready != "" && p.Ready != "<none>" {
				readyInfo = tui.MutedStyle.Render(fmt.Sprintf(" [ready:%s]", p.Ready))
			}

			fmt.Printf("    %s %-45s %s%s%s\n",
				statusStyle.Render(devtools.StatusIcon(p.Status)),
				p.Name,
				statusStyle.Render(p.Status),
				restartInfo,
				readyInfo,
			)
		}
	}
	fmt.Println()
	return nil
}

func selectPod(namespace string) (string, error) {
	pods, err := devtools.ListPods(namespace)
	if err != nil {
		return "", err
	}
	if len(pods) == 0 {
		return "", fmt.Errorf("no pods found in namespace %s", namespace)
	}

	options := make([]huh.Option[string], 0, len(pods))
	for _, p := range pods {
		label := fmt.Sprintf("%s %s (%s)", devtools.StatusIcon(p.Status), p.Name, p.Status)
		options = append(options, huh.NewOption(label, p.Name))
	}

	var selected string
	if err := huh.NewSelect[string]().
		Title("Select pod").
		Options(options...).
		Value(&selected).
		Run(); err != nil {
		return "", nil
	}
	return selected, nil
}

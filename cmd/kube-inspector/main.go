package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"

	"example.com/kube-inspector/internal/check"
	"example.com/kube-inspector/internal/model"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// 退出码：0=扫描完成且本版规则无发现；2=扫描完成且有发现；1=执行失败。
func run(args []string, stdout, stderr io.Writer) int {
	var options model.Options
	flags := flag.NewFlagSet("kube-inspector", flag.ContinueOnError)
	flags.SetOutput(stderr)
	file := flags.String("file", "testdata/pods.json", "path to a Kubernetes PodList JSON file")
	options.RestartThreshold = *flags.Int("restart-threshold", 5, "reported restartCount threshold, inclusive")
	options.PendingThreshold = *flags.Duration("pending-threshold", 5*time.Minute, "reported pending duration threshold, inclusive")
	namespace := flags.String("namespace", "", "only inspect this namespace; empty means all in input")
	format := flags.String("output", "table", "table or json")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}
	if flags.NArg() != 0 || options.RestartThreshold <= 0 || options.PendingThreshold <= 0 || (*format != "table" && *format != "json") {
		fmt.Fprintln(stderr, "invalid arguments: threshold must be positive; output must be table or json; positional arguments are not supported")
		return 1
	}

	pods, err := readPods(*file)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	selected := make([]model.Pod, 0, len(pods))
	for _, pod := range pods {
		if *namespace == "" || pod.Metadata.Namespace == *namespace {
			selected = append(selected, pod)
		}
	}
	findings := check.CheckPods(selected, options)
	if err := writeFindings(stdout, *format, findings); err != nil {
		fmt.Fprintln(stderr, "write report:", err)
		return 1
	}
	fmt.Fprintf(stderr, "input_pods=%d selected_pods=%d findings=%d\n", len(pods), len(selected), len(findings))
	if len(findings) > 0 {
		return 2
	}
	return 0
}

func readPods(path string) ([]model.Pod, error) {
	data, err := os.ReadFile(path) //读取json文件
	if err != nil {
		return nil, fmt.Errorf("read PodList: %w", err)
	}
	var list model.PodList
	if err := json.Unmarshal(data, &list); err != nil { //将json映射为model中的pod对象
		return nil, fmt.Errorf("decode PodList: %w", err)
	}
	if list.Kind != "PodList" || list.Items == nil { //映射完成后就可以使用入list.Kind和list.Items
		return nil, fmt.Errorf("expected kind=PodList and an items array (use kubectl get pods -o json)")
	}
	for i, pod := range list.Items {
		if pod.Metadata.Name == "" || pod.Metadata.Namespace == "" {
			return nil, fmt.Errorf("items[%d]: metadata.name and metadata.namespace are required", i)
		}
	}
	return list.Items, nil
}

func writeFindings(writer io.Writer, format string, findings []check.Finding) error {
	if format == "json" {
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		return encoder.Encode(findings)
	}
	table := tabwriter.NewWriter(writer, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(table, "SEVERITY\tNAMESPACE\tPOD\tCONTAINER\tRULE\tMESSAGE"); err != nil {
		return err
	}
	for _, finding := range findings {
		if _, err := fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\n", finding.Severity,
			finding.Namespace, finding.Pod, finding.Container, finding.Rule, finding.Message); err != nil {
			return err
		}
	}
	return table.Flush()
}

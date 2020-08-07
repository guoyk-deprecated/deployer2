package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	knownWorkloadTypes = []string{
		"deployment",
		"statefulset",
		"daemonset",
		"cronjob",
		"deploy",
		"ds",
		"sts",
	}
)

func convertName(s string) string {
	return strings.TrimSpace(
		strings.ToLower(
			strings.ReplaceAll(
				strings.ReplaceAll(s, ".", "-"),
				"_", "-")))
}

func convertNumber(s string) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		n = -1
	}
	return n
}

type WorkloadOption struct {
	Cluster   string
	Namespace string
	Type      string
	Name      string
	Container string
	IsInit    bool
}

func (w WorkloadOption) String() string {
	sb := &strings.Builder{}
	sb.WriteString(w.Cluster)
	sb.WriteRune('/')
	sb.WriteString(w.Namespace)
	sb.WriteRune('/')
	sb.WriteString(w.Name)
	sb.WriteRune('/')
	sb.WriteString(w.Container)
	if w.IsInit {
		sb.WriteRune('!')
	}
	return sb.String()
}

func (w *WorkloadOption) Set(s string) error {
	splits := strings.Split(s, "/")
	if len(splits) != 4 && len(splits) != 5 {
		return errors.New("目标工作负载参数格式不正确")
	}
	w.Cluster,
		w.Namespace,
		w.Type,
		w.Name = convertName(splits[0]),
		convertName(splits[1]),
		convertName(splits[2]),
		convertName(splits[3])
	if len(splits) == 5 {
		w.Container = convertName(splits[4])
	} else {
		w.Container = w.Name
	}
	w.IsInit = strings.HasSuffix(w.Container, "!")
	w.Container = strings.TrimSuffix(w.Container, "!")
	for _, kt := range knownWorkloadTypes {
		if kt == w.Type {
			return nil
		}
	}
	return errors.New("目标工作负载参数指定了未知的类型")
}

type WorkloadOptions []WorkloadOption

func (ws WorkloadOptions) String() string {
	sb := &strings.Builder{}
	for _, w := range ws {
		if sb.Len() > 0 {
			sb.WriteRune(',')
		}
		sb.WriteString(w.String())
	}
	return sb.String()
}

func (ws *WorkloadOptions) Set(s string) error {
	w := &WorkloadOption{}
	if err := w.Set(s); err != nil {
		return err
	} else {
		*ws = append(*ws, *w)
		return nil
	}
}

type LimitOption struct {
	Min int64
	Max int64
}

func (l LimitOption) IsZero() bool {
	return l.Min == 0 && l.Max == 0
}

func (l LimitOption) String() string {
	return fmt.Sprintf("%d:%d", l.Min, l.Max)
}

func (l *LimitOption) Set(s string) error {
	splits := strings.Split(s, ":")
	if len(splits) != 2 {
		return errors.New("资源配额格式不正确")
	}
	l.Min, l.Max = convertNumber(splits[0]), convertNumber(splits[1])
	if l.Min <= 0 || l.Max <= 0 || l.Max < l.Min {
		return errors.New("资源配额格式不正确")
	}
	return nil
}

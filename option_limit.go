package main

import (
	"errors"
	"fmt"
	"k8s.io/apimachinery/pkg/api/resource"
	"strconv"
	"strings"
)

func decodeLimitNumber(s string) int64 {
	if s == "-" {
		return 0
	}
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		n = -1
	}
	return n
}

type LimitOption struct {
	Min int64
	Max int64
}

func (l *LimitOption) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	var s string
	if err = unmarshal(&s); err != nil {
		return
	}
	if err = l.Set(s); err != nil {
		return
	}
	return
}

func (l LimitOption) IsZero() bool {
	return l.Min == 0 && l.Max == 0
}

func (l LimitOption) String() string {
	if l.Max == 0 {
		return fmt.Sprintf("%d:-", l.Min)
	}
	return fmt.Sprintf("%d:%d", l.Min, l.Max)
}

func (l *LimitOption) Set(s string) error {
	splits := strings.Split(s, ":")
	if len(splits) != 2 {
		return errors.New("资源配额格式不正确")
	}
	l.Min, l.Max = decodeLimitNumber(splits[0]), decodeLimitNumber(splits[1])
	if l.Min <= 0 || l.Max < 0 || (l.Max != 0 && l.Max < l.Min) {
		return errors.New("资源配额格式不正确")
	}
	return nil
}

func (l LimitOption) AsCPU() (resource.Quantity, resource.Quantity) {
	if l.Max == 0 {
		return resource.MustParse(fmt.Sprintf("%dm", l.Min)),
			resource.MustParse(fmt.Sprintf("999"))
	}
	return resource.MustParse(fmt.Sprintf("%dm", l.Min)),
		resource.MustParse(fmt.Sprintf("%dm", l.Max))
}

func (l LimitOption) AsMEM() (resource.Quantity, resource.Quantity) {
	if l.Max == 0 {
		return resource.MustParse(fmt.Sprintf("%dMi", l.Min)),
			resource.MustParse(fmt.Sprintf("999Gi"))
	}
	return resource.MustParse(fmt.Sprintf("%dMi", l.Min)),
		resource.MustParse(fmt.Sprintf("%dMi", l.Max))
}

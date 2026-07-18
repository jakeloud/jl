package entities

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const defaultProxyDelay = 5 * time.Minute

func ParseProjectDomain(value string) (string, time.Duration, error) {
	if value == "" {
		return "", 0, nil
	}
	if strings.Contains(value, "://") || strings.ContainsAny(value, "/?# \t\r\n") {
		return "", 0, fmt.Errorf("invalid project domain %q", value)
	}

	host := value
	delay := defaultProxyDelay
	if separator := strings.LastIndex(value, ":"); separator >= 0 {
		host = value[:separator]
		minutes, err := strconv.Atoi(value[separator+1:])
		if err != nil || minutes < 1 || minutes > 525600 {
			return "", 0, fmt.Errorf("invalid proxy delay in domain %q", value)
		}
		delay = time.Duration(minutes) * time.Minute
	}
	if !validDomainHost(host) {
		return "", 0, fmt.Errorf("invalid project domain %q", value)
	}
	return host, delay, nil
}

func validDomainHost(host string) bool {
	if len(host) == 0 || len(host) > 253 {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, char := range label {
			if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '-' {
				return false
			}
		}
	}
	return true
}

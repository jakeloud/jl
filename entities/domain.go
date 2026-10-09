package entities

import (
	"fmt"
	"strings"
)

func ParseProjectDomain(value string) (string, error) {
	if !validDomainHost(value) {
		return "", fmt.Errorf("invalid project domain %q", value)
	}
	return value, nil
}

func (project Project) ProjectDomains() ([]string, error) {
	values := project.Domain
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		host, err := ParseProjectDomain(value)
		if err != nil {
			return nil, err
		}
		key := strings.ToLower(host)
		if seen[key] {
			return nil, fmt.Errorf("duplicate project domain %q", host)
		}
		seen[key] = true
		result = append(result, host)
	}
	return result, nil
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

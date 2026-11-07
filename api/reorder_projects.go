package api

import (
	"fmt"

	"github.com/jakeloud/jl/entities"
)

func ReorderProjects(params apiRequest) error {
	authenticated, err := entities.IsAuthenticated(params.Email, params.Password)
	if err != nil {
		return fmt.Errorf("authentication check failed: %v", err)
	}
	if !authenticated {
		return fmt.Errorf("authentication required")
	}
	if params.Additional == nil {
		return fmt.Errorf("additional.order is required")
	}

	value, exists := params.Additional["order"]
	if !exists {
		return fmt.Errorf("additional.order is required")
	}
	values, ok := value.([]interface{})
	if !ok {
		return fmt.Errorf("additional.order must be an array of strings")
	}

	conf, err := entities.GetConf()
	if err != nil {
		return err
	}
	projects := make([]entities.Project, 0, len(conf.Projects))
	seen := make(map[string]bool, len(conf.Projects))

	addProject := func(name string) error {
		if seen[name] {
			return fmt.Errorf("project %s appears more than once", name)
		}
		for _, project := range conf.Projects {
			if project.Name == name {
				projects = append(projects, project)
				seen[name] = true
				return nil
			}
		}
		return fmt.Errorf("project %s not found", name)
	}
	err = addProject(entities.JAKELOUD)
	if err != nil {
		return err
	}

	for _, value := range values {
		name, ok := value.(string)
		if !ok {
			return fmt.Errorf("additional.order must contain only strings")
		}
		err = addProject(name)
		if err != nil {
			return err
		}
	}

	if len(conf.Projects) != len(projects) {
		return fmt.Errorf("some projects were missing")
	}

	conf.Projects = projects
	err = entities.SetConf(conf)
	if err != nil {
		return fmt.Errorf("failed to save conf: %w", err)
	}

	return nil
}

package api

import (
	"github.com/jakeloud/jl/entities"
)

func ReorderProjects(params apiRequest) error {
	authenticated, err := entities.IsAuthenticated(params.Email, params.Password)
	if err != nil {
		return fmt.Errorf("authentication check failed: %v", err)
	}
	if !authenticated || paramts.Additional == nil {
		return nil
	}

	value, exists := params.Additional["order"]
	if !exists {
		return nil
	}
	values, ok := value.([]interface{})
	if !ok {
		return fmt.Errorf("additional.order must be an array of strings")
	}

	conf, err := entities.GetConf()
	if err != nil {
		return nil, err
	}
	new_project_list := []entities.Project{}

	func add_to_list(name string) error {
		for _, project := range conf.Projects {
			if project.Name == name {
				new_project_list = append(new_project_list, project)
				return nil
			}
		}
		return fmt.Errorf("project %s not found", name)
	}
	err = add_to_list(entities.JAKELOUD)
	if err != nil {
		return err
	}

	for _, value := range values {
		project_name, ok := value.(string)
		if !ok {
			return fmt.Errorf("additional.order must contain only strings")
		}
		err = add_to_list(project_name)
		if err != nil {
			return err
		}
	}

	if len(conf.Projects) != len(new_project_list) {
		return fmt.Errorf("some projects were missing")
	}
	
	conf.Projects = new_project_list
	err = entities.SetConf(conf)
	if err != nil {
		return fmt.Errorf("failed to save conf")
	}

	return nil
}

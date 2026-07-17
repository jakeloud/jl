package api

import (
	"github.com/jakeloud/jl/entities"
)

func DeleteProject(params apiRequest) error {
	project, err := entities.GetProject(params.Name)
	if err != nil {
		return nil
	}

	authenticated, err := entities.IsAuthenticated(params.Email, params.Password)
	if err != nil {
		return err
	}
	if !authenticated || params.Name == "" {
		return nil
	}

	if err := project.Stop(); err != nil {
		return err
	}

	conf, err := entities.GetConf()
	if err != nil {
		return err
	}

	isRepoUsedElsewhere := false
	count := 0
	for _, p := range conf.Projects {
		if p.Repo == project.Repo {
			count++
			if count > 1 {
				isRepoUsedElsewhere = true
				break
			}
		}
	}

	return project.Remove(!isRepoUsedElsewhere)
}

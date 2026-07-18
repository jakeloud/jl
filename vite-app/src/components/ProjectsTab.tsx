"use client"

import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Plus } from "lucide-react"
import { ProjectCard } from "./ProjectCard"
import { CreateProjectForm } from "./CreateProjectForm"
import type { Project } from "../types"

interface ProjectsTabProps {
  projects?: Project[]
  setSelectedProject: (name: string) => void
  refreshConfig: () => void
}

export function ProjectsTab({ projects = [], setSelectedProject, refreshConfig }: ProjectsTabProps) {
  const [showCreateForm, setShowCreateForm] = useState(false)
  const visibleProjects = projects.filter((project) => project.name !== "jakeloud")

  return (
    <div className="space-y-6">
      {!showCreateForm ? (
        <div className="flex w-full justify-center">
          <Button onClick={() => setShowCreateForm(true)}>
            <Plus className="mr-2 h-4 w-4" /> Add Project
          </Button>
        </div>
      ) : (
        <CreateProjectForm
          onSuccess={() => {
            setShowCreateForm(false)
            refreshConfig()
          }}
          onCancel={() => setShowCreateForm(false)}
        />
      )}

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {visibleProjects.map((project) => (
          <ProjectCard
            key={project.name}
            project={project}
            onSelect={() => setSelectedProject(project.name)}
            refreshConfig={refreshConfig}
          />
        ))}
      </div>

      {visibleProjects.length === 0 && !showCreateForm && (
        <div className="rounded-lg border p-8 text-center">
          <p className="text-muted-foreground">No projects found. Create your first project to get started.</p>
        </div>
      )}
    </div>
  )
}

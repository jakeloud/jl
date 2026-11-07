"use client"

import { useState } from "react"
import { Button } from "@/components/ui/button"
import { ArrowDown, ArrowUp, GripVertical, ListOrdered, LoaderCircle, Plus } from "lucide-react"
import { toast } from "sonner"
import { useApi } from "@/hooks/useApi"
import { ProjectCard } from "./ProjectCard"
import { CreateProjectForm } from "./CreateProjectForm"
import type { Project } from "../types"

interface ProjectsTabProps {
  projects?: Project[]
  setSelectedProject: (name: string) => void
  refreshConfig: () => void | Promise<void>
}

export function ProjectsTab({ projects = [], setSelectedProject, refreshConfig }: ProjectsTabProps) {
  const [showCreateForm, setShowCreateForm] = useState(false)
  const [order, setOrder] = useState<string[] | null>(null)
  const [draggedName, setDraggedName] = useState<string | null>(null)
  const [dropTarget, setDropTarget] = useState<string | null>(null)
  const [isSaving, setIsSaving] = useState(false)
  const { api } = useApi()
  const visibleProjects = projects.filter((project) => project.name !== "jakeloud")
  const currentOrder = visibleProjects.map((project) => project.name)
  const orderChanged = order !== null && order.some((name, index) => name !== currentOrder[index])

  const move = (name: string, target: string) => {
    setOrder((previous) => {
      if (!previous || name === target) return previous
      const from = previous.indexOf(name)
      const to = previous.indexOf(target)
      if (from === -1 || to === -1) return previous
      const next = [...previous]
      next.splice(from, 1)
      next.splice(to, 0, name)
      return next
    })
  }

  const resetDrag = () => {
    setDraggedName(null)
    setDropTarget(null)
  }

  const saveOrder = async () => {
    if (!order || isSaving) return
    setIsSaving(true)
    try {
      const response = await api("reorderProjectsOp", { additional: { order } })
      if (!response.ok) throw new Error("Failed to save project order")
      await refreshConfig()
      setOrder(null)
      toast.success("Project order saved")
    } catch {
      toast.error("Failed to save project order. Please try again.")
    } finally {
      setIsSaving(false)
    }
  }

  return (
    <div className="space-y-6">
      {order !== null ? (
        <section className="overflow-hidden rounded-xl border bg-card shadow-sm" aria-labelledby="reorder-projects-title" aria-busy={isSaving}>
          <div className="space-y-1 border-b bg-muted/40 p-4 sm:p-6">
            <h2 id="reorder-projects-title" className="flex items-center gap-2 font-semibold"><ListOrdered className="size-5 text-muted-foreground" />Reorder projects</h2>
            <p id="reorder-projects-help" className="text-sm text-muted-foreground">Drag a handle onto another row, or use the arrows to move a project. Save to apply your changes.</p>
          </div>
          <ol className="space-y-2 p-4 sm:p-6" aria-describedby="reorder-projects-help">
            {order.map((name, index) => (
              <li
                key={name}
                onDragOver={(event) => {
                  if (!draggedName || isSaving) return
                  event.preventDefault()
                  event.dataTransfer.dropEffect = "move"
                  setDropTarget(name)
                }}
                onDragLeave={(event) => {
                  if (!event.currentTarget.contains(event.relatedTarget as Node | null)) setDropTarget(null)
                }}
                onDrop={(event) => {
                  event.preventDefault()
                  if (draggedName && !isSaving) move(draggedName, name)
                  resetDrag()
                }}
                className={`flex items-center gap-3 rounded-lg border p-3 transition-colors ${draggedName === name ? "border-dashed bg-muted opacity-50" : dropTarget === name ? "border-primary bg-primary/10 ring-2 ring-primary/20" : "bg-background"}`}
              >
                <span
                  draggable={!isSaving}
                  onDragStart={(event) => {
                    event.dataTransfer.effectAllowed = "move"
                    event.dataTransfer.setData("text/plain", name)
                    event.dataTransfer.setDragImage(event.currentTarget.parentElement!, 24, 24)
                    setDraggedName(name)
                  }}
                  onDragEnd={resetDrag}
                  title={`Drag to move ${name}`}
                  className={`rounded p-1 text-muted-foreground ${isSaving ? "cursor-default" : "cursor-grab hover:bg-muted hover:text-foreground active:cursor-grabbing"}`}
                >
                  <GripVertical className="size-5" />
                </span>
                <span className="flex size-7 shrink-0 items-center justify-center rounded-full bg-muted text-xs font-medium tabular-nums text-muted-foreground">{index + 1}</span>
                <span className="min-w-0 flex-1 truncate font-medium" title={name}>{name}</span>
                <div className="flex shrink-0 gap-1">
                  <Button variant="outline" size="icon" className="size-8" aria-label={`Move ${name} up`} disabled={isSaving || index === 0} onClick={() => move(name, order[index - 1])}><ArrowUp className="size-4" /></Button>
                  <Button variant="outline" size="icon" className="size-8" aria-label={`Move ${name} down`} disabled={isSaving || index === order.length - 1} onClick={() => move(name, order[index + 1])}><ArrowDown className="size-4" /></Button>
                </div>
              </li>
            ))}
          </ol>
          <p className="sr-only" aria-live="polite">Current order: {order.join(", ")}</p>
          <div className="flex justify-end gap-2 border-t bg-muted/40 p-4">
            <Button variant="outline" disabled={isSaving} onClick={() => { setOrder(null); resetDrag() }}>Cancel</Button>
            <Button disabled={isSaving || !orderChanged} onClick={saveOrder}>
              {isSaving && <LoaderCircle className="mr-2 size-4 animate-spin" />}
              {isSaving ? "Saving..." : "Save order"}
            </Button>
          </div>
        </section>
      ) : !showCreateForm ? (
        <div className="flex w-full flex-wrap justify-center gap-2">
          <Button onClick={() => setShowCreateForm(true)}>
            <Plus className="mr-2 h-4 w-4" /> Add Project
          </Button>
          {visibleProjects.length > 1 && (
            <Button variant="outline" onClick={() => setOrder(currentOrder)}>
              <ListOrdered className="mr-2 size-4" /> Reorder projects
            </Button>
          )}
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

      {order === null && <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {visibleProjects.map((project) => (
          <ProjectCard
            key={project.name}
            project={project}
            onSelect={() => setSelectedProject(project.name)}
            refreshConfig={refreshConfig}
          />
        ))}
      </div>}

      {visibleProjects.length === 0 && !showCreateForm && (
        <div className="rounded-lg border p-8 text-center">
          <p className="text-muted-foreground">No projects found. Create your first project to get started.</p>
        </div>
      )}
    </div>
  )
}

"use client"

import { useEffect, useState } from "react"
import { Button } from "@/components/ui/button"
import type { Project } from "../types"
import { getDomainFavicon } from "@/lib/favicon"
import { defaultProjectCommand, isValidProjectHost, parseProjectDomain } from "@/lib/projects"
import { Label } from "@/components/ui/label"
import { Skeleton } from "@/components/ui/skeleton"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { ArrowLeft, ExternalLink, Globe, RefreshCw, Settings2, Trash2 } from "lucide-react"
import { useApi } from "@/hooks/useApi"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { toast } from "sonner"
import { CommandsEditor, DomainsEditor } from "@/components/ListEditors"

interface ProjectViewProps {
  project: Project
  back: () => void
  refreshConfig: () => void
}

type ProjectResponse = Project | { message: string }

export function ProjectView({ project: initialProject, back, refreshConfig }: ProjectViewProps) {
  const [project, setProject] = useState(initialProject)
  const initialCommands = initialProject.additional?.cmd?.length ? initialProject.additional.cmd : defaultProjectCommand(initialProject.name).split("\n")
  const [commands, setCommands] = useState(initialCommands)
  const [useDefaultCommand, setUseDefaultCommand] = useState(initialCommands.join("\n") === defaultProjectCommand(initialProject.name))
  const [favicon, setFavicon] = useState("")
  const [faviconLoading, setFaviconLoading] = useState(Boolean(initialProject.domain?.length))
  const [isRefreshing, setIsRefreshing] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)
  const [isRebooting, setIsRebooting] = useState(false)
  const [isUpdatingDomain, setIsUpdatingDomain] = useState(false)
  const [showDomainEditor, setShowDomainEditor] = useState(false)
  const initialDomainValues = initialProject.domain || []
  const [domains, setDomains] = useState(initialDomainValues.map((value) => parseProjectDomain(value).host))
  const { api } = useApi()

  const domainValues = project.domain || []
  const domain = parseProjectDomain(domainValues[0])
  const runtime = project.additional?.runtime
  const state = project.state || "unknown"
  const currentRelease = project.additional?.currentRelease

  useEffect(() => {
    setProject(initialProject)
  }, [initialProject])

  useEffect(() => {
    const persistedCommands = project.additional?.cmd?.length ? project.additional.cmd : defaultProjectCommand(project.name).split("\n")
    setCommands(persistedCommands)
    setUseDefaultCommand(persistedCommands.join("\n") === defaultProjectCommand(project.name))
  }, [project.additional?.cmd, project.name])

  useEffect(() => {
    if (!domain.enabled) {
      setFavicon("")
      setFaviconLoading(false)
      return
    }
    setFaviconLoading(true)
    getDomainFavicon(domainValues[0] || "").then((url) => {
      setFavicon(url)
      setFaviconLoading(false)
    })
  }, [domain.enabled, domainValues[0]])

  const getProject = async () => {
    setIsRefreshing(true)
    try {
      const response = await api("getAppOp", { name: project.name })
      const data = await response.json() as ProjectResponse
      if ("message" in data) throw new Error(data.message)
      setProject(data)
    } catch {
      toast.error("Failed to fetch project")
    } finally {
      setIsRefreshing(false)
    }
  }

  const redeploy = async (nextDomains = domainValues) => {
    await api("createAppOp", {
      name: project.name,
      domain: nextDomains,
      repo: project.repo || "",
      additional: { cmd: useDefaultCommand ? [] : commands.map((line) => line.trim()).filter(Boolean) },
    })
  }

  const handleReboot = async () => {
    setIsRebooting(true)
    try {
      await redeploy()
      toast.success("Project reboot initiated")
      await refreshConfig()
      await getProject()
    } catch {
      toast.error("Failed to reboot project")
    } finally {
      setIsRebooting(false)
    }
  }

  const handleDomainUpdate = async () => {
    if (!domains.length || domains.some((host) => !isValidProjectHost(host)) || new Set(domains.map((host) => host.toLowerCase())).size !== domains.length) {
      toast.error("Enter unique valid domains")
      return
    }
    setIsUpdatingDomain(true)
    try {
      await redeploy(domains)
      setShowDomainEditor(false)
      toast.success("Domain update initiated")
      await refreshConfig()
      await getProject()
    } catch {
      toast.error("Failed to update domain")
    } finally {
      setIsUpdatingDomain(false)
    }
  }

  const handleDelete = async () => {
    setIsDeleting(true)
    try {
      await api("deleteAppOp", { name: project.name })
      toast.success("Project deleted")
      await refreshConfig()
      back()
    } catch {
      toast.error("Failed to delete project")
    } finally {
      setIsDeleting(false)
    }
  }

  const openDomainEditor = () => {
    const values = project.domain || []
    setDomains(values.map((item) => parseProjectDomain(item).host))
    setShowDomainEditor(true)
  }


  return (
    <div className="container mx-auto max-w-5xl p-2">
      <Button onClick={back}><ArrowLeft /> Back</Button>

      <Card className="m-6">
        <CardHeader>
          <div className="flex flex-col justify-between gap-3 sm:flex-row">
            <div>
              <CardTitle className="flex items-center gap-2">
                {faviconLoading ? <Skeleton className="size-8 rounded-full" /> : (
                  <Avatar><AvatarImage src={favicon} /><AvatarFallback><Globe /></AvatarFallback></Avatar>
                )}
                {project.name}
              </CardTitle>
              <CardDescription className="mt-1 flex items-center gap-2">
                {domainValues.length ? domainValues.map((value) => {
                  const item = parseProjectDomain(value)
                  return <a key={item.host} href={`https://${item.host}`} target="_blank" rel="noopener noreferrer" className="mr-1 inline-flex items-center gap-1 rounded-full border bg-muted px-2 py-0.5 text-xs hover:underline">{item.host}<ExternalLink className="size-3" /></a>
                }) : "No domain"}
                <span className="relative">
                  <Button variant="ghost" size="icon" className="size-7" onClick={openDomainEditor}><Settings2 className="size-4" /></Button>
                  {showDomainEditor && (
                    <div className="absolute left-0 top-9 z-20 w-80 space-y-3 rounded-md border bg-popover p-4 text-popover-foreground shadow-md">
                      <div className="space-y-1"><Label>Domains</Label><DomainsEditor value={domains} onChange={setDomains} /></div>
                      <div className="flex justify-end gap-2">
                        <Button size="sm" variant="outline" onClick={() => setShowDomainEditor(false)}>Cancel</Button>
                        <Button size="sm" onClick={handleDomainUpdate} disabled={isUpdatingDomain}>{isUpdatingDomain ? "Saving..." : "Save and redeploy"}</Button>
                      </div>
                    </div>
                  )}
                </span>
              </CardDescription>
            </div>
            <div className="flex flex-col gap-2">
              <Button size="sm" onClick={getProject} disabled={isRefreshing}>
                <RefreshCw /> {isRefreshing ? "Updating..." : "Update status"}
              </Button>
              <Button variant="outline" size="sm" onClick={handleReboot} disabled={isRebooting}>
                <RefreshCw /> {isRebooting ? "Rebooting..." : "Full Reboot"}
              </Button>
            </div>
          </div>
        </CardHeader>
      </Card>

      <Card className="m-6">
        <CardHeader><CardTitle>Status / Logs</CardTitle></CardHeader>
        <CardContent>
          <div className="flex items-center gap-2">
            <pre className="min-w-0 flex-1 overflow-x-auto rounded-md bg-muted p-4">{state.startsWith("Error") ? `Error: ${state.slice(6).trim()}` : state}</pre>
          </div>
          {project.additional?.ps && <pre className="mt-4 overflow-x-auto rounded-md bg-muted p-4">{project.additional.ps}</pre>}
          {project.additional?.logs && <pre className="mt-4 max-h-[36rem] overflow-auto rounded-md bg-muted p-4">{project.additional.logs}</pre>}
        </CardContent>
      </Card>

      <Card className="m-6">
        <CardHeader><CardTitle>Project</CardTitle></CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-3 text-sm sm:grid-cols-2">
            <div><span className="text-muted-foreground">Port</span><div className="font-mono">{project.port ?? "Not allocated"}</div></div>
            <div><span className="text-muted-foreground">Release</span><div className="font-mono">{currentRelease ?? "Not created"}</div></div>
            <div><span className="text-muted-foreground">Process</span><div className="font-mono">{runtime?.pid ? `PID ${runtime.pid}` : "Not running"}</div></div>
            <div><span className="text-muted-foreground">Role</span><div>{runtime?.active ? "Active" : runtime?.alive ? "Candidate" : "Inactive"}</div></div>
          </div>
          <div><span className="text-sm text-muted-foreground">Repository</span><div className="break-all font-mono text-sm">{project.repo || "Not set"}</div></div>
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={useDefaultCommand}
              onChange={(event) => {
                setUseDefaultCommand(event.target.checked)
                if (event.target.checked) setCommands(defaultProjectCommand(project.name).split("\n"))
              }}
              className="size-4"
            />
            Use default command on next reboot
          </label>
          <div className="space-y-2">
            <Label>Build and start commands</Label>
            <div className={useDefaultCommand ? "pointer-events-none opacity-50" : ""}><CommandsEditor value={commands} onChange={setCommands} /></div>
            <p className="text-sm text-muted-foreground">JakeLoud provides $PORT.</p>
          </div>
        </CardContent>
      </Card>

      <AlertDialog>
        <AlertDialogTrigger asChild>
          <div className="m-6 flex justify-center">
            <Button variant="destructive" disabled={isDeleting}><Trash2 className="h-4 w-4" />{isDeleting ? "Deleting..." : "Delete Project"}</Button>
          </div>
        </AlertDialogTrigger>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete this project?</AlertDialogTitle>
            <AlertDialogDescription>This permanently removes "{project.name}" and its project directory.</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={handleDelete}>Delete</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

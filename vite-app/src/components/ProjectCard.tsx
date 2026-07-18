"use client"

import { useEffect, useState } from "react"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Badge } from "@/components/ui/badge"
import { Ellipsis, ExternalLink, Globe, RefreshCw } from "lucide-react"
import type { Project } from "@/types"
import { useApi } from "@/hooks/useApi"
import { getDomainFavicon } from "@/lib/favicon"
import { parseProjectDomain } from "@/lib/projects"
import { toast } from "sonner"

interface ProjectCardProps {
  project: Project
  onSelect: () => void
  refreshConfig: () => void
}

export function ProjectCard({ project, onSelect, refreshConfig }: ProjectCardProps) {
  const domain = parseProjectDomain(project.domain)
  const [isRebooting, setIsRebooting] = useState(false)
  const [favicon, setFavicon] = useState("")
  const [faviconLoading, setFaviconLoading] = useState(domain.enabled)
  const { api } = useApi()

  const handleReboot = async () => {
    setIsRebooting(true)
    try {
      await api("createAppOp", {
        name: project.name,
        domain: project.domain || "",
        repo: project.repo || "",
        additional: { cmd: project.additional?.cmd || "" },
      })
      toast.success("Project reboot initiated")
      refreshConfig()
    } catch {
      toast.error("Failed to reboot project")
    } finally {
      setIsRebooting(false)
    }
  }

  const state = project.state || "unknown"
  const stateShort = state.startsWith("Error") ? "Error" : state

  useEffect(() => {
    if (!domain.enabled) {
      setFavicon("")
      setFaviconLoading(false)
      return
    }
    setFaviconLoading(true)
    getDomainFavicon(project.domain || "").then((url) => {
      setFavicon(url)
      setFaviconLoading(false)
    })
  }, [domain.enabled, project.domain])

  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between gap-3">
          <div>
            <CardTitle className="flex items-center gap-2">
              {faviconLoading ? (
                <Skeleton className="size-8 rounded-full" />
              ) : (
                <Avatar>
                  <AvatarImage src={favicon} />
                  <AvatarFallback><Globe /></AvatarFallback>
                </Avatar>
              )}
              {project.name}
            </CardTitle>
            <CardDescription>
              {domain.enabled ? (
                <a href={`https://${domain.host}`} target="_blank" rel="noopener noreferrer" className="flex items-center hover:underline">
                  {domain.host} <ExternalLink className="ml-1 h-3 w-3" />
                </a>
              ) : "No domain"}
              {domain.enabled && <span className="ml-2">Switch after {domain.timeoutMinutes} min</span>}
            </CardDescription>
          </div>
          <Badge variant={state === "🟢 running" ? "default" : "secondary"}>{stateShort}</Badge>
        </div>
      </CardHeader>
      <CardContent className="flex-1 text-sm">
        <span className="font-medium">Repository:</span> {project.repo || "Not set"}
      </CardContent>
      <CardFooter className="flex justify-end gap-2">
        <Button variant="outline" size="sm" onClick={handleReboot} disabled={isRebooting}>
          <RefreshCw className="mr-2 h-4 w-4" />
          {isRebooting ? "Rebooting..." : "Full Reboot"}
        </Button>
        <Button size="icon" onClick={onSelect} className="size-8">
          <Ellipsis className="h-4 w-4" />
        </Button>
      </CardFooter>
    </Card>
  )
}

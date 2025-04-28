"use client"

import { useState, useEffect } from "react"
import { Button } from "@/components/ui/button"
import { App } from "../types"
import { getDomainFavicon } from "@/lib/favicon"
import { Skeleton } from "@/components/ui/skeleton"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { ArrowLeft, Globe, ExternalLink, RefreshCw, Trash2 } from "lucide-react"
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

interface StatusProps {
  ps: string
}
function Status({ps}: StatusProps) {
  let out
  try {
    const j = JSON.parse(ps)
    out = j.Status
  } catch(e) {
    out = ps
  }
  return (
    <pre className="mt-4 p-4 bg-muted rounded-md overflow-x-auto">
      {out}
    </pre>
  )
}

interface AppsViewProps {
  app: App
  back: () => void
  refreshConfig: () => void
}
export function AppView({ app: initialApp, back, refreshConfig }: AppsViewProps) {
  const [app, setApp] = useState(initialApp)
  const [favicon, setFavicon] = useState("")
  const [faviconLoading, setFaviconLoading] = useState(true)
  const [isDeleting, setIsDeleting] = useState(false)
  const [isRebooting, setIsRebooting] = useState(false)
  const { api } = useApi()

  useEffect(() => {
    getApp()
  }, [])

  const handleReboot = async () => {
    setIsRebooting(true)
    try {
      await api("createAppOp", app)
      toast.success("App reboot initiated")
      refreshConfig()
    } catch (error) {
      toast.error("Failed to reboot app")
    } finally {
      setIsRebooting(false)
    }
  }
  const handleDelete = async () => {
    setIsDeleting(true)
    try {
      await api("deleteAppOp", app)
      toast.success("App deletion initiated")
      refreshConfig()
    } catch (error) {
      toast.error("Failed to delete app")
    } finally {
      setIsDeleting(false)
    }
  }


  const getApp = async () => {
    try {
      const response = await api(
        "getAppOp",
        {name: app.name},
      )
      const data = await response.json()
      setApp(data)
    } catch (error) {
      toast.error("Failed to fetch app")
    }
  }

  const logs = `${app.state.startsWith('Error') ? '🔴 ' : ''}${app.state}`

  const dockerOptions = app.additional?.dockerOptions || '<empty>'

  const additionalLogs = app.additional?.logs || ''
  const ps = app.additional?.ps

  useEffect(() => {
    getDomainFavicon(app.domain).then(url => {
      setFavicon(url)
      setFaviconLoading(false)
    })
  }, [app.domain])

  return (
    <div className="max-w-5xl container mx-auto p-2">
      <Button onClick={back}>
        <ArrowLeft/>
        Back
      </Button>
      <Card className="m-6">
        <CardHeader>
        <div className="flex flex-col sm:flex-row justify-between gap-2">
          <div>
            <CardTitle className="flex items-center gap-2">
              {faviconLoading ? (
                <Skeleton className="rounded-full size-8"/>
              ) : (
                <Avatar>
                  <AvatarImage src={favicon} />
                  <AvatarFallback><Globe/></AvatarFallback>
                </Avatar>
              )}
              {app.name}
            </CardTitle>
            <CardDescription>
              <a
                href={`https://${app.domain}`}
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center hover:underline"
              >
                {app.domain} <ExternalLink className="ml-1 h-3 w-3" />
              </a>
            </CardDescription>
          </div>
          <div className="flex flex-col gap-2">
            <Button size="sm" onClick={getApp}>
              <RefreshCw/>
              Update status
            </Button>

            <Button variant="outline" size="sm" onClick={handleReboot} disabled={isRebooting}>
              <RefreshCw className="mr-2 h-4 w-4" />
              {isRebooting ? "Rebooting..." : "Full Reboot"}
            </Button>
          </div>
        </div>
        </CardHeader>
      </Card>

      <Card className="m-6">
        <CardHeader>
          <CardTitle>
            Status / Logs
          </CardTitle>
        </CardHeader>
        <CardContent>
          <pre className="p-4 bg-muted rounded-md overflow-x-auto">
            {logs}
          </pre>
          {ps ? <Status ps={ps}/> : null}
          {additionalLogs != undefined ? (
            <pre className="mt-4 p-4 bg-muted rounded-md overflow-x-auto">
              {additionalLogs}
            </pre>
          ) : null}
        </CardContent>
      </Card>

      <Card className="m-6">
        <CardHeader>
          <CardTitle className="flex max-w-sm justify-between gap-2">
            <span className="text-muted-foreground">Port</span>
            <span className="wrap-anywhere font-mono">{app.port}</span>
          </CardTitle>
        </CardHeader>
        <CardHeader>
          <CardTitle className="flex max-w-sm justify-between gap-2">
            <span className="text-muted-foreground">Repo</span>
            <span className="wrap-anywhere font-mono">{app.repo}</span>
          </CardTitle>
        </CardHeader>
        <CardHeader>
          <CardTitle>
            Docker Options
          </CardTitle>
        </CardHeader>
        <CardContent>
          <pre className="p-4 bg-muted rounded-md overflow-x-auto">
            {dockerOptions}
          </pre>
        </CardContent>
      </Card>


      <AlertDialog>
        <AlertDialogTrigger asChild>
          <div className="m-6 flex justify-center">
            <Button variant="destructive" disabled={isDeleting}>
              <Trash2 className="h-4 w-4" />
              {isDeleting ? "Deleting..." : "Delete App"}
            </Button>
          </div>
        </AlertDialogTrigger>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Are you absolutely sure?</AlertDialogTitle>
            <AlertDialogDescription>
              This action cannot be undone. This will permanently delete the app "{app.name}" and remove all
              associated data.
            </AlertDialogDescription>
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

"use client"

import { useState, useEffect } from "react"
import { Card, CardContent, CardHeader, CardTitle, CardDescription, CardFooter } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Badge } from "@/components/ui/badge"
import { Ellipsis, Globe, ExternalLink, RefreshCw } from "lucide-react"
import { App } from "@/types"
import { useApi } from "@/hooks/useApi"
import { toast } from "sonner"
import { getDomainFavicon } from "@/lib/favicon"

interface AppCardProps {
  app: App
  onSelect: () => void
  refreshConfig: () => void
}
export function AppCard({ app, onSelect, refreshConfig }: AppCardProps) {
  const [isRebooting, setIsRebooting] = useState(false)
  const [favicon, setFavicon] = useState("")
  const [faviconLoading, setFaviconLoading] = useState(true)
  const { api } = useApi()

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

  const stateShort = app.state.startsWith('Error') ? '🔴 Error' : app.state

  useEffect(() => {
    getDomainFavicon(app.domain).then(url => {
      setFavicon(url)
      setFaviconLoading(false)
    })
  }, [app.domain])

  return (
    <Card>
      <CardHeader>
        <div className="flex justify-between items-start">
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
          <Badge variant={app.state === "running" ? "default" : "secondary"}>{stateShort}</Badge>
        </div>
      </CardHeader>
      <CardContent className="flex-1">
        <div className="space-y-2 text-sm">
          <div>
            <span className="font-medium">Repository:</span> {app.repo}
          </div>
          {app.dockerOptions && (
            <div>
              <span className="font-medium">Docker Options:</span> {app.dockerOptions}
            </div>
          )}
        </div>
      </CardContent>
      <CardFooter className="flex justify-end space-x-2">
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

"use client"

import { useState } from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { ExternalLink, Copy } from "lucide-react"
import { Project } from "../types"
import { useApi } from "../hooks/useApi"
import { toast } from "sonner"
import { DomainsEditor } from "@/components/ListEditors"
import { parseProjectDomain } from "@/lib/projects"
interface ChangeDomainProps {
  jakeloudApp: Project
}
function ChangeDomain({ jakeloudApp }: ChangeDomainProps) {
  const [isLoading, setIsLoading] = useState(false)
  const { api } = useApi()
  const initialDomains = jakeloudApp.domain || []
  const [domains, setDomains] = useState(initialDomains.map((value) => parseProjectDomain(value).host))

  async function onSubmit() {
    if (!domains.length) { toast.error("Add at least one domain"); return }
    setIsLoading(true)
    try {
      await api("setJakeloudDomainOp", { domain: domains })
      window.location.replace(`https://${domains[0]}`)
    } catch {
      toast.error("Failed to set domain")
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Change Domain (Jakeloud dashboard)</CardTitle>
      </CardHeader>
      <CardContent className="space-y-6">
        <DomainsEditor value={domains} onChange={setDomains} placeholder="example.com" />
        <Button type="button" onClick={onSubmit} disabled={isLoading}>{isLoading ? "Assigning..." : "Save domains"}</Button>
      </CardContent>
    </Card>
  )
}


interface SSHKeyProps {
  sshKey: string
}
function SSHKey({sshKey}: SSHKeyProps) {
  const copySshKeyToClipboard = () => {
    navigator.clipboard.writeText(sshKey.trim())
    toast.success("SSH key copied to clipboard")
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>SSH Key</CardTitle>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="space-y-2">
          <div className="relative">
            <pre className="p-4 bg-muted rounded-md overflow-x-auto">
              {sshKey || "No SSH key available"}
            </pre>
            {sshKey && (
              <Button
                variant="outline"
                size="sm"
                className="absolute top-2 right-2"
                onClick={copySshKeyToClipboard}
              >
                <Copy className="h-4 w-4" />
              </Button>
            )}
          </div>
        </div>
        <p className="flex">
          Go to
            <a
              href={`https://github.com/settings/keys`}
              className="ml-1 underline flex items-center mr-2"
              target="_blank"
            >
              github.com/settings/keys
              <ExternalLink
                className="mt-1 ml-0.5 h-3 w-3"
              />
            </a>
          and create
            <span className="ml-1 tracking-tight font-semibold">New SSH key</span>
          .
        </p>
      </CardContent>
    </Card>
  )
}


interface SettingsTabProps {
  apps?: Project[]
  refreshConfig: () => void
}
export function SettingsTab({ apps = [] }: SettingsTabProps) {
  const jakeloudApp = apps.find((app) => app.name === "jakeloud")

  if (!jakeloudApp) {
    return <div>Loading settings...</div>
  }

  return (
    <div className="space-y-6">
      <h2 className="text-xl font-semibold">Jakeloud Settings</h2>
      <SSHKey sshKey={jakeloudApp.additional?.sshKey || ''}/>
      <ChangeDomain jakeloudApp={jakeloudApp}/>
    </div>
  )
}

import { useState } from "react"
import { ChevronDown, Copy, ExternalLink } from "lucide-react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Label } from "@/components/ui/label"

export function AgentSkillCard() {
  const [method, setMethod] = useState("npx")
  const [installation, setInstallation] = useState("global")
  const command = `${method} skills add jakeloud/skill${installation === "global" ? " -g -a codex -a opencode" : ""}`

  const copyCommand = async () => {
    try {
      await navigator.clipboard.writeText(command)
      toast.success("Installation command copied")
    } catch {
      toast.error("Could not copy the command. Select and copy it manually.")
    }
  }

  return (
    <Card className="mt-6">
      <CardHeader>
        <CardTitle>Use JakeLoud with your agent</CardTitle>
        <CardDescription>
          Install the JakeLoud skill to help your coding agent work with JakeLoud.
        </CardDescription>
        <Button asChild variant="link" className="h-auto justify-self-start p-0">
          <a href="https://jakeloud.com/" target="_blank" rel="noopener noreferrer">
            Read the documentation
            <ExternalLink aria-hidden="true" />
          </a>
        </Button>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="skill-install-method">Installation method</Label>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button id="skill-install-method" variant="outline" className="w-full justify-between">
                  {method}
                  <ChevronDown aria-hidden="true" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start" className="min-w-[var(--radix-dropdown-menu-trigger-width)]">
                <DropdownMenuRadioGroup value={method} onValueChange={setMethod}>
                  <DropdownMenuRadioItem value="npx">npx</DropdownMenuRadioItem>
                  <DropdownMenuRadioItem value="bunx">bunx</DropdownMenuRadioItem>
                </DropdownMenuRadioGroup>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
          <div className="space-y-2">
            <Label htmlFor="skill-install-agent">Agent installation</Label>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button id="skill-install-agent" variant="outline" className="w-full justify-between">
                  {installation === "global" ? "Codex + OpenCode (global)" : "Interactive installation"}
                  <ChevronDown aria-hidden="true" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start" className="min-w-[var(--radix-dropdown-menu-trigger-width)]">
                <DropdownMenuRadioGroup value={installation} onValueChange={setInstallation}>
                  <DropdownMenuRadioItem value="global">Codex + OpenCode (global)</DropdownMenuRadioItem>
                  <DropdownMenuRadioItem value="interactive">Interactive installation</DropdownMenuRadioItem>
                </DropdownMenuRadioGroup>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
        <div className="flex flex-col gap-3 rounded-md border bg-muted p-3 sm:flex-row sm:items-center">
          <code className="min-w-0 flex-1 break-all text-sm select-all">{command}</code>
          <Button type="button" variant="outline" onClick={copyCommand}>
            <Copy aria-hidden="true" />
            Copy command
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

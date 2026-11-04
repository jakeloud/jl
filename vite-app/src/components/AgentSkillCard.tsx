import { useState } from "react"
import { ChevronDown, Copy, ExternalLink } from "lucide-react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader } from "@/components/ui/card"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Label } from "@/components/ui/label"

const agents: Record<string, string> = {
  codex: "Codex",
  opencode: "OpenCode",
  "claude-code": "Claude Code",
  cursor: "Cursor",
  "github-copilot": "GitHub Copilot",
  "gemini-cli": "Gemini CLI",
  windsurf: "Windsurf",
  cline: "Cline",
  amp: "Amp",
  roo: "Roo Code",
}

export function AgentSkillCard() {
  const [method, setMethod] = useState("npx")
  const [installation, setInstallation] = useState("interactive")
  const command = `${method} skills add jakeloud/skill${installation === "interactive" ? "" : ` -g -a ${installation} -y`}`

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
        <Button asChild size="lg" className="h-12 justify-between text-base">
          <a href="https://jakeloud.com/" target="_blank" rel="noopener noreferrer">
            Documentation
            <ExternalLink aria-hidden="true" />
          </a>
        </Button>
      </CardHeader>
      <CardContent className="space-y-3 border-t pt-4">
        <h2 className="text-sm font-medium text-muted-foreground">Install agent skill</h2>
        <div className="flex flex-wrap gap-2">
          <div>
            <Label htmlFor="skill-install-method" className="sr-only">Installation method</Label>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button id="skill-install-method" variant="outline" size="sm" className="w-24 justify-between">
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
          <div className="flex min-w-0 items-center gap-2">
            <Label htmlFor="skill-install-agent">agent:</Label>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button id="skill-install-agent" variant="outline" size="sm" className="w-full justify-between">
                  {installation === "interactive" ? "Interactive installation" : `${agents[installation]} (global)`}
                  <ChevronDown aria-hidden="true" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start" className="min-w-[var(--radix-dropdown-menu-trigger-width)]">
                <DropdownMenuRadioGroup value={installation} onValueChange={setInstallation}>
                  <DropdownMenuLabel>Global installation</DropdownMenuLabel>
                  {Object.entries(agents).map(([value, label]) => (
                    <DropdownMenuRadioItem key={value} value={value}>{label}</DropdownMenuRadioItem>
                  ))}
                  <DropdownMenuSeparator />
                  <DropdownMenuRadioItem value="interactive">Interactive installation</DropdownMenuRadioItem>
                </DropdownMenuRadioGroup>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
        <div className="flex items-center gap-2 rounded-md border bg-muted p-2">
          <code className="min-w-0 flex-1 break-all text-sm select-all">{command}</code>
          <Button type="button" variant="ghost" size="icon" aria-label="Copy command" onClick={copyCommand}>
            <Copy aria-hidden="true" />
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

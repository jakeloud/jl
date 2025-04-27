"use client"

import { useState } from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Input } from "@/components/ui/input"
import { useForm } from "react-hook-form"
import { z } from "zod"
import { zodResolver } from "@hookform/resolvers/zod"
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form"
import { ExternalLink, Copy, RefreshCw } from "lucide-react"
import { App } from "../types"
import { useApi } from "../hooks/useApi"
import { toast } from "sonner"

const domainFormSchema = z.object({
  domain: z.string().min(3, { message: "Domain must be at least 3 characters" }),
})
interface ChangeDomainProps {
  jakeloudApp: App
}
function ChangeDomain({ jakeloudApp }: ChangeDomainProps) {
  const [isLoading, setIsLoading] = useState(false)
  const { api } = useApi()
  const initialDomain = jakeloudApp.domain

  const form = useForm<z.infer<typeof domainFormSchema>>({
    resolver: zodResolver(domainFormSchema),
    defaultValues: {
      domain: initialDomain,
    },
  })

  async function onSubmit(values: z.infer<typeof domainFormSchema>) {
    if (values.domain == initialDomain) {
      return
    }
    setIsLoading(true)
    try {
      await api("setJakeloudDomainOp", values)
      window.location.replace(`https://${values.domain}`)
    } catch (error) {
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
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="domain"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Domain</FormLabel>
                  <FormControl>
                    <Input
                      placeholder="example.com" {...field}
                      className="max-w-md"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <Button type="submit" disabled={isLoading}>
              {isLoading ? "Assigning..." : "Assign Domain"}
            </Button>
          </form>
        </Form>
      </CardContent>
    </Card>
  )
}


interface SSHKeyProps {
  sshKey: string
}
function SSHKey({sshKey}: SSHKeyProps) {
  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
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
                onClick={() => copyToClipboard(sshKey)}
              >
                <Copy className="h-4 w-4" />
              </Button>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}


const telegramFormSchema = z.object({
  chatId: z.string().min(1, { message: "Chat ID is required" }),
  botToken: z.string().min(1, { message: "Bot Token is required" }),
})
interface TelegramIntegrationProps {
  jakeloudApp: App,
  refreshConfig: () => void
}
function TelegramIntegration({jakeloudApp, refreshConfig}: TelegramIntegrationProps) {
  const [isUpdatingTelegram, setIsUpdatingTelegram] = useState(false)
  const { api } = useApi()

  const additional = jakeloudApp?.additional || {}

  const form = useForm<z.infer<typeof telegramFormSchema>>({
    resolver: zodResolver(telegramFormSchema),
    defaultValues: {
      chatId: additional.chatId || "",
      botToken: additional.botToken || "",
    },
  })

  const onSubmitTelegramForm = async (values: z.infer<typeof telegramFormSchema>) => {
    setIsUpdatingTelegram(true)
    try {
      await api("setJakeloudAdditionalOp", {
        additional: { ...additional, ...values },
      })
      toast.success("Telegram settings updated")
      refreshConfig()
    } catch (error) {
      toast.error("Failed to update Telegram settings")
    } finally {
      setIsUpdatingTelegram(false)
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Telegram Integration</CardTitle>
      </CardHeader>
      <CardContent className="space-y-6">
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmitTelegramForm)} className="space-y-4">
            <FormField
              control={form.control}
              name="chatId"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Chat ID</FormLabel>
                  <FormControl>
                    <Input
                      {...field}
                      className="max-w-md"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <p className="flex">
              To get your Chat ID visit this
              <a
                href={`https://api.telegram.org/bot<token>/getUpdates`}
                className="ml-1 underline flex items-center mr-2"
                target="_blank"
              >
                page
                <ExternalLink
                  className="mt-1 ml-0.5 h-3 w-3"
                />
              </a>
              and replace &lt;token&gt; with your Bot Token.
            </p>
            <FormField
              control={form.control}
              name="botToken"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Bot Token</FormLabel>
                  <FormControl>
                    <Input
                      {...field}
                      type="password"
                      className="max-w-md"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <Button type="submit" disabled={isUpdatingTelegram}>
              {isUpdatingTelegram ? "Updating..." : "Update Telegram Settings"}
            </Button>
          </form>
        </Form>
      </CardContent>
    </Card>
  )
}

interface DockerCacheProps {
  refreshConfig: () => void
}
function DockerCache({refreshConfig}: DockerCacheProps) {
  const [isClearingCache, setIsClearingCache] = useState(false)
  const { api } = useApi()

  const handleClearCache = async () => {
    setIsClearingCache(true)
    try {
      await api("clearCacheOp")
      toast.success("Cache cleared successfully")
      refreshConfig()
    } catch (error) {
      toast.error("Failed to clear cache")
    } finally {
      setIsClearingCache(false)
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Docker Cache</CardTitle>
      </CardHeader>
      <CardContent className="space-y-6">
        <div>
          <Button variant="outline" onClick={handleClearCache} disabled={isClearingCache}>
            <RefreshCw className="mr-2 h-4 w-4" />
            {isClearingCache ? "Clearing..." : "Clear Cache"}
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

interface SettingsTabProps {
  apps?: App[]
  refreshConfig: () => void
}
export function SettingsTab({ apps = [], refreshConfig }: SettingsTabProps) {
  const jakeloudApp = apps.find((app) => app.name === "jakeloud")

  if (!jakeloudApp) {
    return <div>Loading settings...</div>
  }

  return (
    <div className="space-y-6">
      <h2 className="text-xl font-semibold">Jakeloud Settings</h2>
      <SSHKey sshKey={jakeloudApp.additional?.sshKey || ''}/>
      <TelegramIntegration
        jakeloudApp={jakeloudApp}
        refreshConfig={refreshConfig}
      />
      <DockerCache refreshConfig={refreshConfig}/>
      <ChangeDomain jakeloudApp={jakeloudApp}/>
    </div>
  )
}

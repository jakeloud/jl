"use client"

import { useState } from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Switch } from "@/components/ui/switch"
import { Label } from "@/components/ui/label"
import { Input } from "@/components/ui/input"
import { useForm } from "react-hook-form"
import { z } from "zod"
import { zodResolver } from "@hookform/resolvers/zod"
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form"
import { Copy, RefreshCw } from "lucide-react"
import type { App } from "../types"
import { useApi } from "../hooks/useApi"
import { toast } from "sonner"

interface SettingsTabProps {
  apps?: App[]
  refreshConfig: () => void
}

const telegramFormSchema = z.object({
  chatId: z.string().min(1, { message: "Chat ID is required" }),
  botToken: z.string().min(1, { message: "Bot Token is required" }),
})

export function SettingsTab({ apps = [], refreshConfig }: SettingsTabProps) {
  const [isClearingCache, setIsClearingCache] = useState(false)
  const [isUpdatingTelegram, setIsUpdatingTelegram] = useState(false)
  const { api } = useApi()

  const jakeloudApp = apps.find((app) => app.name === "jakeloud")
  const additional = jakeloudApp?.additional || {}

  const form = useForm<z.infer<typeof telegramFormSchema>>({
    resolver: zodResolver(telegramFormSchema),
    defaultValues: {
      chatId: additional.chatId || "",
      botToken: additional.botToken || "",
    },
  })

  const handleRegisterAllowedChange = async (checked: boolean) => {
    try {
      await api("setJakeloudAdditionalOp", {
        additional: { ...additional, registerAllowed: checked },
      })
      toast.success("Registration settings updated")
      refreshConfig()
    } catch (error) {
      toast.error("Failed to update registration settings")
    }
  }

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

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
    toast.success("SSH key copied to clipboard")
  }

  if (!jakeloudApp) {
    return <div>Loading settings...</div>
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Jakeloud Settings</CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="flex items-center space-x-2">
            <Switch
              id="registration-allowed"
              checked={additional.registerAllowed === true}
              onCheckedChange={handleRegisterAllowedChange}
            />
            <Label htmlFor="registration-allowed">Allow Registration</Label>
          </div>

          <div className="space-y-2">
            <h3 className="text-lg font-medium">SSH Key</h3>
            <div className="relative">
              <pre className="p-4 bg-muted rounded-md overflow-x-auto">
                {additional.sshKey || "No SSH key available"}
              </pre>
              {additional.sshKey && (
                <Button
                  variant="outline"
                  size="sm"
                  className="absolute top-2 right-2"
                  onClick={() => copyToClipboard(additional.sshKey || "")}
                >
                  <Copy className="h-4 w-4" />
                </Button>
              )}
            </div>
          </div>

          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmitTelegramForm)} className="space-y-4">
              <h3 className="text-lg font-medium">Telegram Integration</h3>
              <FormField
                control={form.control}
                name="chatId"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Chat ID</FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <p>
                To get your Chat ID visit this
                <a
                  href={`https://api.telegram.org/bot<token>/getUpdates`}
                  className="ml-1 underline"
                  target="_blank"
                >page</a>
              </p>
              <FormField
                control={form.control}
                name="botToken"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Bot Token</FormLabel>
                    <FormControl>
                      <Input {...field} type="password" />
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

          <div>
            <Button variant="outline" onClick={handleClearCache} disabled={isClearingCache}>
              <RefreshCw className="mr-2 h-4 w-4" />
              {isClearingCache ? "Clearing..." : "Clear Cache"}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

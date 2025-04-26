"use client"

import { useState, useEffect } from "react"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Button } from "@/components/ui/button"
import { toast } from 'sonner'
import { Toaster } from "@/components/ui/sonner"
import { LoginForm } from "./components/LoginForm"
import { RegisterForm } from "./components/RegisterForm"
import { DomainForm } from "./components/DomainForm"
import { AppsTab } from "./components/AppsTab"
import { SettingsTab } from "./components/SettingsTab"
import { useAuth } from "./hooks/useAuth"
import { useApi } from "./hooks/useApi"
import type { AppConfig } from "./types"

function App() {
  const [config, setConfig] = useState<AppConfig | null>(null)
  const [activeTab, setActiveTab] = useState<string>("apps")
  const { isAuthenticated, logout } = useAuth()
  const { api } = useApi()

  const getConfig = async () => {
    try {
      const response = await api("getConfOp")
      const data = await response.json()
      setConfig(data)
    } catch (error) {
      toast.error("Failed to fetch configuration")
    }
  }

  useEffect(() => {
    getConfig()
  }, [isAuthenticated])

  if (!config) {
    return <div className="flex items-center justify-center h-screen">Loading...</div>
  }

  if (config.message === "domain") {
    return <DomainForm onSuccess={getConfig} />
  }

  if (config.message === "login") {
    return (
      <div className="container mx-auto max-w-md p-6 space-y-6">
        <Tabs defaultValue="login">
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="login">Login</TabsTrigger>
            <TabsTrigger value="register">Register</TabsTrigger>
          </TabsList>
          <TabsContent value="login">
            <LoginForm onSuccess={getConfig} />
          </TabsContent>
          <TabsContent value="register">
            <RegisterForm onSuccess={getConfig} />
          </TabsContent>
        </Tabs>
      </div>
    )
  }

  if (config.message === "register") {
    return <RegisterForm onSuccess={getConfig} />
  }

  return (
    <div className="container mx-auto p-4">
      <header className="mb-6">
        <div className="flex justify-between items-center mb-4">
          <h1 className="text-2xl font-bold">Jakeloud Dashboard</h1>
          <Button variant="outline" onClick={logout}>
            Logout
          </Button>
        </div>
        <Tabs value={activeTab} onValueChange={setActiveTab}>
          <TabsList>
            <TabsTrigger value="apps">Apps</TabsTrigger>
            <TabsTrigger value="settings">Settings</TabsTrigger>
          </TabsList>
        </Tabs>
      </header>

      <main>
        {activeTab === "apps" && <AppsTab apps={config.apps} refreshConfig={getConfig} />}
        {activeTab === "settings" && <SettingsTab apps={config.apps} refreshConfig={getConfig} />}
      </main>
      <Toaster />
    </div>
  )
}

export default App

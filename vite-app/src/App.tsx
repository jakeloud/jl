"use client"

import { useState, useEffect } from "react"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { toast } from 'sonner'
import { LoginForm } from "@/components/LoginForm"
import { RegisterForm } from "@/components/RegisterForm"
import { ProjectsTab } from "@/components/ProjectsTab"
import { ProjectView } from "@/components/ProjectView"
import { DBView } from "@/components/DBView"
import { SettingsTab } from "@/components/SettingsTab"
import { DBsTab } from "@/components/DBsTab"
import { useAuth } from "@/hooks/useAuth"
import { useApi } from "@/hooks/useApi"
import { JakeLoudConfig } from "@/types"

function App() {
  const [config, setConfig] = useState<JakeLoudConfig | null>(null)
  const [activeTab, setActiveTab] = useState<string>("projects")
  const [selectedProject, setSelectedProject] = useState('')
  const [selectedDB, setSelectedDB] = useState('')
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

  if (config.message === "login") {
    return (
      <div className="w-full h-dvh flex items-center justify-center">
        <div className="w-full container mx-auto max-w-md p-6 space-y-6">
          <LoginForm onSuccess={getConfig} />
        </div>
      </div>
    )
  }

  if (config.message === "register") {
    return (
      <div className="w-full h-dvh flex items-center justify-center">
        <RegisterForm onSuccess={getConfig} />
      </div>
    )
  }

  const project = config?.apps?.find(project => project.name == selectedProject)
  if (project != undefined) {
    return (
      <ProjectView
        project={project}
        refreshConfig={getConfig}
        back={() => setSelectedProject('')}
      />
    )
  }

  const db = (config?.dbs || []).find(d => d.name == selectedDB)
  if (db != undefined) {
    return (
      <DBView
        db={db}
        refreshConfig={getConfig}
        back={() => setSelectedDB('')}
      />
    )
  }

  return (
    <div className="max-w-5xl container mx-auto p-2">
      <header className="mb-6">
        <div className="grid grid-cols-3 items-center mb-4">
          <div className="flex items-center">
            <img src="/favicon.png" className="size-6"/>
            <h1 className="text-lg font-bold">JakeLoud</h1>
          </div>
          <div className="flex justify-center">
            <Tabs value={activeTab} onValueChange={setActiveTab}>
              <TabsList>
                <TabsTrigger value="projects">Projects</TabsTrigger>
                <TabsTrigger value="settings">Settings</TabsTrigger>
                <TabsTrigger value="dbs">DBs</TabsTrigger>
              </TabsList>
            </Tabs>
          </div>
          <div className="flex justify-end">
            <Button variant="outline" onClick={logout}>
              Logout
            </Button>
          </div>
        </div>
        <Separator/>
      </header>

      <main>
        {activeTab === "projects" && (
          <ProjectsTab
            projects={config.apps}
            refreshConfig={getConfig}
            setSelectedProject={setSelectedProject}
          />
        )}
        {activeTab === "settings" && <SettingsTab apps={config.apps} refreshConfig={getConfig} />}
        {activeTab === "dbs" && <DBsTab dbs={config.dbs || []} refreshConfig={getConfig} setSelectedDB={setSelectedDB} />}
      </main>
    </div>
  )
}

export default App

"use client"

import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Plus } from "lucide-react"
import { AppCard } from "./AppCard"
import { CreateAppForm } from "./CreateAppForm"
import type { App } from "../types"

interface AppsTabProps {
  apps?: App[]
  refreshConfig: () => void
}

export function AppsTab({ apps = [], refreshConfig }: AppsTabProps) {
  const [showCreateForm, setShowCreateForm] = useState(false)

  const filteredApps = apps.filter((app) => app.name !== "jakeloud")

  return (
    <div className="space-y-6">

      {!showCreateForm ? (
        <div className="w-full flex justify-center">
          <Button onClick={() => setShowCreateForm(true)}>
            <Plus className="mr-2 h-4 w-4" /> Add App
          </Button>
        </div>
      ) : (
        <CreateAppForm
          onSuccess={() => {
            setShowCreateForm(false)
            refreshConfig()
          }}
          onCancel={() => setShowCreateForm(false)}
        />
      )}
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {filteredApps.map((app) => (
          <AppCard key={app.name} app={app} refreshConfig={refreshConfig} />
        ))}
      </div>

      {filteredApps.length === 0 && !showCreateForm && (
        <div className="text-center p-8 border rounded-lg">
          <p className="text-muted-foreground">No apps found. Create your first app to get started.</p>
        </div>
      )}
    </div>
  )
}

"use client"

import { useState } from 'react'
import { Button } from "@/components/ui/button"
import { DB } from "../types"
import { Plus } from "lucide-react"
import { CreateDBForm } from "./CreateDBForm"
import { DBCard } from "./DBCard"


interface DBsTabProps {
  dbs?: DB[]
  refreshConfig: () => void
  setSelectedDB: (name: string) => void
}
export function DBsTab({ dbs = [], refreshConfig, setSelectedDB }: DBsTabProps) {
  const [showCreateForm, setShowCreateForm] = useState(false)

  return (
    <div className="space-y-6">
      <h2 className="text-xl font-semibold">Databases explorer</h2>

      {!showCreateForm ? (
        <div className="w-full flex justify-center">
          <Button onClick={() => setShowCreateForm(true)}>
            <Plus className="mr-2 h-4 w-4" /> Add DB connection
          </Button>
        </div>
      ) : (
        <CreateDBForm
          onSuccess={() => {
            setShowCreateForm(false)
            refreshConfig()
          }}
          onCancel={() => setShowCreateForm(false)}
        />
      )}

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {dbs.map((db) => (
          <DBCard
            key={db.name}
            db={db}
            onSelect={() => setSelectedDB(db.name)}
            refreshConfig={refreshConfig}
          />
        ))}
      </div>

      {dbs.length === 0 && !showCreateForm && (
        <div className="text-center p-8 border rounded-lg">
          <p className="text-muted-foreground">No DB connections found. Create your first db connection to get started.</p>
        </div>
      )}
    </div>
  )
}

"use client"

import { DB } from "../types"


interface DBsTabProps {
  dbs?: DB[]
  refreshConfig: () => void
}
export function DBsTab({ dbs = [], refreshConfig }: DBsTabProps) {
  return (
    <div className="space-y-6">
      <h2 className="text-xl font-semibold">Databases explorer</h2>
    </div>
  )
}

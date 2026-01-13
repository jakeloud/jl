"use client"

import { useState, useEffect } from "react"
import { Button } from "@/components/ui/button"
import { DB } from "../types"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { ArrowLeft, Database, RefreshCw, Trash2 } from "lucide-react"
import { useApi } from "@/hooks/useApi"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { toast } from "sonner"

interface DBViewProps {
  db: DB
  back: () => void
  refreshConfig: () => void
}
export function DBView({ db, back, refreshConfig }: DBViewProps) {
  const [isDeleting, setIsDeleting] = useState(false)
  const [selectedTable, setSelectedTable] = useState('')
  const [tables, setTables] = useState([])
  const [rows, setRows] = useState([])
  const [total, setTotal] = useState(0)
  const { api } = useApi()

  const queryDB = async () => {
    try {
      const response = await api(
        "queryDBOp",
        {name: db.name, table: selectedTable},
      )
      const data = await response.json()
      setTables(data.tables || [])
      setRows(data.rows || [])
      setTotal(data.total || 0)
    } catch (error) {
      toast.error("Failed to query database")
    }
  }

  useEffect(() => {
    queryDB()
  }, [])

  const handleDelete = async () => {
    setIsDeleting(true)
    try {
      await api("deleteDBConnectionOp", db)
      toast.success("DB connection deletion initiated")
      refreshConfig()
    } catch (error) {
      toast.error("Failed to delete db connection")
    } finally {
      setIsDeleting(false)
    }
  }


  return (
    <div className="max-w-5xl container mx-auto p-2">
      <Button onClick={back}>
        <ArrowLeft/>
        Back
      </Button>
      <Card className="m-6">
        <CardHeader>
        <div className="flex flex-col sm:flex-row justify-between gap-2">
          <div>
            <CardTitle className="flex items-center gap-2">
                <Avatar>
                  <AvatarFallback><Database/></AvatarFallback>
                </Avatar>
              {db.name}
            </CardTitle>
            <CardDescription>
                {db.path}
            </CardDescription>
          </div>
          <div className="flex flex-col gap-2">
            <Button size="sm" onClick={queryDB}>
              <RefreshCw/>
              Update status
            </Button>
          </div>
        </div>
        </CardHeader>
      </Card>

        
    {tables.length > 0 ? (
            <Tabs value={selectedTable} onValueChange={setSelectedTable}>
              <TabsList>
                {tables.map((table) => (
                        <TabsTrigger key={table} value={table}>{table}</TabsTrigger>
                ))}
              </TabsList>
            </Tabs>
    ) : null}

      <Card className="m-6">
        <CardHeader>
          <CardTitle>
            Total: {total}
          </CardTitle>
        </CardHeader>
        <CardContent>
        {rows.map((row, index) => (
                        <div key={index}>
                                {JSON.stringify(row)}
                        </div>
        ))}
        </CardContent>
      </Card>


      <AlertDialog>
        <AlertDialogTrigger asChild>
          <div className="m-6 flex justify-center">
            <Button variant="destructive" disabled={isDeleting}>
              <Trash2 className="h-4 w-4" />
              {isDeleting ? "Deleting..." : "Delete DB connection"}
            </Button>
          </div>
        </AlertDialogTrigger>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Are you absolutely sure?</AlertDialogTitle>
            <AlertDialogDescription>
              This action can be undone. This will NOT delete actual database.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={handleDelete}>Delete</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

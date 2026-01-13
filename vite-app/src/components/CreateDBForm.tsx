"use client"

import { useState } from "react"
import { useForm } from "react-hook-form"
import { z } from "zod"
import { zodResolver } from "@hookform/resolvers/zod"
import { Button } from "@/components/ui/button"
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form"
import { Input } from "@/components/ui/input"
import { useApi } from "@/hooks/useApi"
import { toast } from "sonner"

const formSchema = z.object({
  name: z.string().regex(/^[a-zA-Z0-9][a-zA-Z0-9_.-]+$/, {
    message: 'Must start with an alphanumeric character and contain only alphanumeric characters, underscores, dots, or hyphens',
  }),
  path: z.string().min(1, { message: "Path to sqlite database is required" }),
})

interface CreateDBFormProps {
  onSuccess: () => void
  onCancel: () => void
}

export function CreateDBForm({ onSuccess, onCancel }: CreateDBFormProps) {
  const [isLoading, setIsLoading] = useState(false)
  const { api } = useApi()

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: "",
      path: "",
    },
  })

  async function onSubmit(values: z.infer<typeof formSchema>) {
    setIsLoading(true)
    try {
      await api("createDBConnectionOp", values)
      toast.success("DB connection creating.")
      onSuccess()
    } catch (error) {
      toast.error("Failed to create DB connection")
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="space-y-4 p-4 border rounded-lg">
      <h2 className="text-lg font-semibold">Create New App</h2>
      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
          <FormField
            control={form.control}
            name="name"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Database Name</FormLabel>
                <FormControl>
                  <Input placeholder="my-database" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="path"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Path</FormLabel>
                <FormControl>
                  <Input placeholder="/app/db/my-database.sqlite" {...field} />
                </FormControl>
                <FormDescription>Path to sqlite database file. Example: "/app/db/my-database.sqlite"</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <div className="flex justify-end space-x-2">
            <Button type="button" variant="outline" onClick={onCancel}>
              Cancel
            </Button>
            <Button type="submit" disabled={isLoading}>
              {isLoading ? "Creating..." : "Create DB connection"}
            </Button>
          </div>
        </form>
      </Form>
    </div>
  )
}

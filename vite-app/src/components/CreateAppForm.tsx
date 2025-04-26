"use client"

import { useState } from "react"
import { useForm } from "react-hook-form"
import { z } from "zod"
import { zodResolver } from "@hookform/resolvers/zod"
import { Button } from "@/components/ui/button"
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { useApi } from "../hooks/useApi"
import { toast } from "sonner"

const formSchema = z.object({
  name: z.string().min(1, { message: "App name is required" }),
  domain: z.string().min(3, { message: "Domain must be at least 3 characters" }),
  repo: z.string().min(1, { message: "Repository URL is required" }),
  dockerOptions: z.string().optional(),
})

interface CreateAppFormProps {
  onSuccess: () => void
  onCancel: () => void
}

export function CreateAppForm({ onSuccess, onCancel }: CreateAppFormProps) {
  const [isLoading, setIsLoading] = useState(false)
  const { api } = useApi()

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: "",
      domain: "",
      repo: "",
      dockerOptions: "",
    },
  })

  async function onSubmit(values: z.infer<typeof formSchema>) {
    setIsLoading(true)
    try {
      await api("createAppOp", values)
      toast.success("App creation initiated. Refresh to track progress.")
      onSuccess()
    } catch (error) {
      toast.error("Failed to create app")
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
                <FormLabel>App Name</FormLabel>
                <FormControl>
                  <Input placeholder="my-app" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="domain"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Domain</FormLabel>
                <FormControl>
                  <Input placeholder="app.example.com" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="repo"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Repository</FormLabel>
                <FormControl>
                  <Input placeholder="git@github.com:user/repo.git" {...field} />
                </FormControl>
                <FormDescription>
                  Enter GitHub repo in format "git@github.com:user/repo.git" (as seen in SSH clone option)
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="dockerOptions"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Docker Options</FormLabel>
                <FormControl>
                  <Textarea placeholder="-v /home/jakeloud:/home/jakeloud -e PASSWORD=jakeloud" {...field} />
                </FormControl>
                <FormDescription>Example: "-v /home/jakeloud:/home/jakeloud -e PASSWORD=jakeloud"</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <div className="flex justify-end space-x-2">
            <Button type="button" variant="outline" onClick={onCancel}>
              Cancel
            </Button>
            <Button type="submit" disabled={isLoading}>
              {isLoading ? "Creating..." : "Create App"}
            </Button>
          </div>
        </form>
      </Form>
    </div>
  )
}

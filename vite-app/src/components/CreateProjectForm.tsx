"use client"

import { useEffect, useState } from "react"
import { useForm } from "react-hook-form"
import { z } from "zod"
import { zodResolver } from "@hookform/resolvers/zod"
import { Button } from "@/components/ui/button"
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { useApi } from "@/hooks/useApi"
import { defaultProjectCommand, formatProjectDomain, isValidProjectHost } from "@/lib/projects"
import { toast } from "sonner"

const formSchema = z.object({
  name: z.string().regex(/^[a-zA-Z0-9][a-zA-Z0-9_.-]*$/, {
    message: "Use letters, numbers, underscores, dots, or hyphens",
  }),
  domainEnabled: z.boolean(),
  domainHost: z.string(),
  timeoutMinutes: z.number().int().min(1).max(525600),
  repo: z.string().min(1, { message: "Repository URL is required" }),
  useDefaultCommand: z.boolean(),
  cmd: z.string(),
}).superRefine((values, context) => {
  if (values.domainEnabled && !isValidProjectHost(values.domainHost)) {
    context.addIssue({ code: "custom", path: ["domainHost"], message: "Enter a valid hostname" })
  }
  if (!values.useDefaultCommand && !values.cmd.trim()) {
    context.addIssue({ code: "custom", path: ["cmd"], message: "Command is required" })
  }
})

type ProjectForm = z.infer<typeof formSchema>

interface CreateProjectFormProps {
  onSuccess: () => void
  onCancel: () => void
}

export function CreateProjectForm({ onSuccess, onCancel }: CreateProjectFormProps) {
  const [isLoading, setIsLoading] = useState(false)
  const { api } = useApi()
  const form = useForm<ProjectForm>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: "",
      domainEnabled: false,
      domainHost: "",
      timeoutMinutes: 5,
      repo: "",
      useDefaultCommand: true,
      cmd: defaultProjectCommand(""),
    },
  })

  const name = form.watch("name")
  const domainEnabled = form.watch("domainEnabled")
  const useDefaultCommand = form.watch("useDefaultCommand")

  useEffect(() => {
    if (useDefaultCommand) {
      form.setValue("cmd", defaultProjectCommand(name))
    }
  }, [form, name, useDefaultCommand])

  async function onSubmit(values: ProjectForm) {
    setIsLoading(true)
    try {
      await api("createAppOp", {
        name: values.name,
        repo: values.repo,
        domain: formatProjectDomain(values.domainEnabled, values.domainHost, values.timeoutMinutes),
        additional: { cmd: values.useDefaultCommand ? "" : values.cmd },
      })
      toast.success("Project creation initiated. Refresh to track progress.")
      onSuccess()
    } catch {
      toast.error("Failed to create project")
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="space-y-4 rounded-lg border p-4">
      <h2 className="text-lg font-semibold">Create New Project</h2>
      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
          <FormField control={form.control} name="name" render={({ field }) => (
            <FormItem>
              <FormLabel>Project Name</FormLabel>
              <FormControl><Input placeholder="my-project" {...field} /></FormControl>
              <FormMessage />
            </FormItem>
          )} />

          <FormField control={form.control} name="repo" render={({ field }) => (
            <FormItem>
              <FormLabel>Repository</FormLabel>
              <FormControl><Input placeholder="git@github.com:user/repo.git" {...field} /></FormControl>
              <FormDescription>Use the SSH clone URL from your repository.</FormDescription>
              <FormMessage />
            </FormItem>
          )} />

          <FormField control={form.control} name="domainEnabled" render={({ field }) => (
            <FormItem className="flex items-center gap-2">
              <FormControl>
                <input type="checkbox" checked={field.value} onChange={field.onChange} className="size-4" />
              </FormControl>
              <FormLabel className="font-normal">Enable domain and proxy</FormLabel>
            </FormItem>
          )} />

          {domainEnabled && (
            <div className="grid gap-4 sm:grid-cols-[1fr_12rem]">
              <FormField control={form.control} name="domainHost" render={({ field }) => (
                <FormItem>
                  <FormLabel>Domain</FormLabel>
                  <FormControl><Input placeholder="project.example.com" {...field} /></FormControl>
                  <FormMessage />
                </FormItem>
              )} />
              <FormField control={form.control} name="timeoutMinutes" render={({ field }) => (
                <FormItem>
                  <FormLabel>Liveness timeout</FormLabel>
                  <FormControl>
                    <Input type="number" min={1} max={525600} value={field.value} onChange={(event) => field.onChange(event.target.valueAsNumber)} />
                  </FormControl>
                  <FormDescription>Minutes</FormDescription>
                  <FormMessage />
                </FormItem>
              )} />
            </div>
          )}

          <FormField control={form.control} name="useDefaultCommand" render={({ field }) => (
            <FormItem className="flex items-center gap-2">
              <FormControl>
                <input type="checkbox" checked={field.value} onChange={field.onChange} className="size-4" />
              </FormControl>
              <FormLabel className="font-normal">Use default command</FormLabel>
            </FormItem>
          )} />

          <FormField control={form.control} name="cmd" render={({ field }) => (
            <FormItem>
              <FormLabel>Build and start command</FormLabel>
              <FormControl><Textarea className="min-h-28 font-mono" disabled={useDefaultCommand} {...field} /></FormControl>
              <FormDescription>JakeLoud runs this from the release directory and provides $PORT.</FormDescription>
              <FormMessage />
            </FormItem>
          )} />

          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={onCancel}>Cancel</Button>
            <Button type="submit" disabled={isLoading}>{isLoading ? "Creating..." : "Create Project"}</Button>
          </div>
        </form>
      </Form>
    </div>
  )
}

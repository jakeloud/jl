"use client"

import { useLayoutEffect, useRef, useState } from "react"
import { GripVertical, Minus, Plus } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"

interface DomainsEditorProps {
  value: string[]
  onChange: (domains: string[]) => void
  placeholder?: string
}

export function DomainsEditor({ value, onChange, placeholder = "www.example.com" }: DomainsEditorProps) {
  const [draft, setDraft] = useState("")
  const [removeIndex, setRemoveIndex] = useState<number | null>(null)
  const add = () => {
    const domain = draft.trim()
    if (!domain || value.some((item) => item.toLowerCase() === domain.toLowerCase())) return
    onChange([...value, domain])
    setDraft("")
  }

  return <div className="space-y-3">
    <div className="flex flex-wrap gap-2">
      {value.map((domain, index) => <span key={`${domain}-${index}`} className="inline-flex items-center gap-1 rounded-full border bg-muted px-3 py-1 text-sm">
        {domain}
        <Button type="button" variant="ghost" size="icon" className="size-6 rounded-full" aria-label={`Remove ${domain}`} onClick={() => setRemoveIndex(index)}><Minus className="size-3" /></Button>
      </span>)}
      {!value.length && <span className="text-sm text-muted-foreground">No domains added</span>}
    </div>
    <div className="flex gap-2">
      <Input value={draft} placeholder={placeholder} onChange={(event) => setDraft(event.target.value)} onKeyDown={(event) => { if (event.key === "Enter") { event.preventDefault(); add() } }} />
      <Button type="button" variant="outline" onClick={add} disabled={!draft.trim()}><Plus className="mr-1 size-4" />Add domain</Button>
    </div>
    <AlertDialog open={removeIndex !== null} onOpenChange={(open) => { if (!open) setRemoveIndex(null) }}>
      <AlertDialogContent>
        <AlertDialogHeader><AlertDialogTitle>Remove this domain?</AlertDialogTitle><AlertDialogDescription>{removeIndex !== null ? value[removeIndex] : ""} will be removed from this project.</AlertDialogDescription></AlertDialogHeader>
        <AlertDialogFooter><AlertDialogCancel>Cancel</AlertDialogCancel><AlertDialogAction onClick={() => { if (removeIndex !== null) onChange(value.filter((_, index) => index !== removeIndex)); setRemoveIndex(null) }}>Remove domain</AlertDialogAction></AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
}

interface CommandsEditorProps {
  value: string[]
  onChange: (commands: string[]) => void
}

export function CommandsEditor({ value, onChange }: CommandsEditorProps) {
  const [draft, setDraft] = useState("")
  const [removeIndex, setRemoveIndex] = useState<number | null>(null)
  const [dragIndex, setDragIndex] = useState<number | null>(null)
  const ids = useRef(value.map(() => crypto.randomUUID()))
  const rowElements = useRef(new Map<string, HTMLLIElement>())
  const previousPositions = useRef(new Map<string, DOMRect>())
  const [, refreshIds] = useState(0)
  const update = (next: string[]) => {
    next.forEach((_, index) => { if (!ids.current[index]) ids.current[index] = crypto.randomUUID() })
    ids.current.length = next.length
    onChange(next)
  }
  const move = (from: number, to: number) => {
    if (from === to) return
    previousPositions.current = new Map([...rowElements.current].map(([id, element]) => [id, element.getBoundingClientRect()]))
    const next = [...value]
    const [item] = next.splice(from, 1)
    next.splice(to, 0, item)
    const [id] = ids.current.splice(from, 1)
    ids.current.splice(to, 0, id)
    onChange(next)
    refreshIds((current) => current + 1)
  }
  useLayoutEffect(() => {
    for (const [id, element] of rowElements.current) {
      const before = previousPositions.current.get(id)
      if (!before) continue
      const after = element.getBoundingClientRect()
      const x = before.left - after.left
      const y = before.top - after.top
      if (x || y) element.animate([{ transform: `translate(${x}px, ${y}px)` }, { transform: "translate(0, 0)" }], { duration: 180, easing: "ease-out" })
    }
    previousPositions.current.clear()
  })
  const add = () => {
    if (!draft.trim()) return
    update([...value, draft.trim()])
    setDraft("")
  }
  return <div className="space-y-3">
    <ol className="divide-y rounded-md border">
      {value.map((command, index) => <li key={ids.current[index]} ref={(element) => { const id = ids.current[index]; if (element) rowElements.current.set(id, element); else rowElements.current.delete(id) }} draggable onDragStart={() => setDragIndex(index)} onDragOver={(event) => event.preventDefault()} onDrop={() => { if (dragIndex !== null) move(dragIndex, index); setDragIndex(null) }} onDragEnd={() => setDragIndex(null)} className={`flex items-center gap-2 px-2 py-1.5 ${dragIndex === index ? "bg-muted opacity-60" : ""}`}>
        <GripVertical className="size-4 shrink-0 cursor-grab text-muted-foreground" aria-label="Drag to reorder" />
        <span className="w-6 shrink-0 text-xs text-muted-foreground">{index + 1}</span>
        <Input value={command} onChange={(event) => onChange(value.map((item, itemIndex) => itemIndex === index ? event.target.value : item))} className="h-9 min-w-0 flex-1 font-mono" aria-label={`Command ${index + 1}`} />
        {index === value.length - 1 && <span title="JakeLoud promotes the release only if this final command starts and remains running." className="hidden shrink-0 cursor-help rounded-full bg-primary/10 px-2 py-0.5 text-xs text-primary sm:inline">Liveness check ⓘ</span>}
        <Button type="button" variant="ghost" size="icon" aria-label={`Remove command ${index + 1}`} onClick={() => setRemoveIndex(index)}><Minus className="size-4" /></Button>
      </li>)}
    </ol>
    <div className="flex gap-2">
      <Input value={draft} placeholder="Add a shell command" onChange={(event) => setDraft(event.target.value)} onKeyDown={(event) => { if (event.key === "Enter") { event.preventDefault(); add() } }} />
      <Button type="button" variant="outline" onClick={add} disabled={!draft.trim()}><Plus className="mr-1 size-4" />Add command</Button>
    </div>
    <p className="text-sm text-muted-foreground">Drag commands to change their order. JakeLoud monitors the last command; hover over its label for details.</p>
    <AlertDialog open={removeIndex !== null} onOpenChange={(open) => { if (!open) setRemoveIndex(null) }}>
      <AlertDialogContent>
        <AlertDialogHeader><AlertDialogTitle>Remove this command?</AlertDialogTitle><AlertDialogDescription>{removeIndex !== null ? value[removeIndex] : ""}</AlertDialogDescription></AlertDialogHeader>
        <AlertDialogFooter><AlertDialogCancel>Cancel</AlertDialogCancel><AlertDialogAction onClick={() => { if (removeIndex !== null) { ids.current.splice(removeIndex, 1); update(value.filter((_, index) => index !== removeIndex)) } setRemoveIndex(null) }}>Remove command</AlertDialogAction></AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
}

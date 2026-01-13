"use client"

import { Card, CardHeader, CardTitle, CardDescription, CardFooter } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Ellipsis, Database } from "lucide-react"
import { DB } from "@/types"

interface DBCardProps {
  db: DB
  onSelect: () => void
}
export function DBCard({ db, onSelect }: DBCardProps) {
  return (
    <Card>
      <CardHeader>
        <div className="flex justify-between items-start">
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
        </div>
      </CardHeader>
      <CardFooter className="flex justify-end space-x-2">
        <Button size="icon" onClick={onSelect} className="size-8">
          <Ellipsis className="h-4 w-4" />
        </Button>
      </CardFooter>
    </Card>
  )
}

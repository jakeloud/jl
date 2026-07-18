"use client"

import { useAuth } from "./useAuth"
import type { FormData, Project } from "../types"

type T = Project | FormData
export function useApi() {
  const { getLoginData } = useAuth()

  const api = async (op: string, body: T = {}) => {
    return await fetch("/api", {
      method: "POST",
      body: JSON.stringify({ op, ...getLoginData(), ...body }),
    })
  }

  return { api }
}

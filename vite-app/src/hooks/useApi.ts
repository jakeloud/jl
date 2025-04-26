"use client"

import { useAuth } from "./useAuth"
import type { FormData } from "../types"

export function useApi() {
  const { getLoginData } = useAuth()

  const api = async (op: string, body: FormData = {}) => {
    return await fetch("/api", {
      method: "POST",
      body: JSON.stringify({ op, ...getLoginData(), ...body }),
    })
  }

  return { api }
}

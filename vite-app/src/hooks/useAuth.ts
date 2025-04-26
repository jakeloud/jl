"use client"

import { useState, useEffect } from "react"
import type { LoginData } from "../types"

export function useAuth() {
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(false)

  useEffect(() => {
    const { email, password } = getLoginData()
    setIsAuthenticated(!!(email && password))
  }, [])

  const setLoginData = (password: string, email: string) => {
    window.localStorage.setItem("pwd", password)
    window.localStorage.setItem("email", email)
    setIsAuthenticated(!!(email && password))
  }

  const getLoginData = (): LoginData => {
    const password = window.localStorage.getItem("pwd") || ""
    const email = window.localStorage.getItem("email") || ""
    return { password, email }
  }

  const logout = () => {
    window.localStorage.removeItem("pwd")
    window.localStorage.removeItem("email")
    setIsAuthenticated(false)
    window.location.reload()
  }

  return {
    isAuthenticated,
    setLoginData,
    getLoginData,
    logout,
  }
}

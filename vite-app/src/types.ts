export interface ReleaseRuntime {
  release: number
  pid?: number
  alive: boolean
  active: boolean
  promotionDeadline?: string
}

export interface ProjectAdditional {
  cmd?: string
  currentRelease?: number
  runtime?: ReleaseRuntime
  promotionDeadline?: string
  ps?: string
  logs?: string
  registerAllowed?: boolean
  chatId?: string
  botToken?: string
  sshKey?: string
}

export interface Project {
  name: string
  domain?: string
  repo?: string
  email?: string
  state?: string
  port?: number
  additional?: ProjectAdditional
}

export interface JakeLoudConfig {
  message?: "login" | "register"
  apps?: Project[]
}

export interface LoginData {
  email: string
  password: string
}

export interface FormData {
  [key: string]: any
}

export interface App {
  name: string
  domain: string
  repo: string
  email: string
  state: string
  port?: string
  dockerOptions?: string
  additional?: {
    dockerOptions?: string
    registerAllowed?: boolean
    chatId?: string
    botToken?: string
    sshKey?: string
    ps?: string
    logs?: string
  }
}

export interface DB {
        name: string
        path: string
}

export interface AppConfig {
  message?: "login" | "register"
  apps?: App[]
  dbs?: DB[]
}

export interface LoginData {
  email: string
  password: string
}

export interface FormData {
  [key: string]: any
}

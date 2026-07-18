export const DEFAULT_LIVENESS_TIMEOUT = 5

export interface ProjectDomain {
  enabled: boolean
  host: string
  timeoutMinutes: number
}

export function parseProjectDomain(value = ""): ProjectDomain {
  if (!value) {
    return { enabled: false, host: "", timeoutMinutes: DEFAULT_LIVENESS_TIMEOUT }
  }

  const separator = value.lastIndexOf(":")
  if (separator > -1) {
    const timeout = Number(value.slice(separator + 1))
    if (Number.isInteger(timeout) && timeout > 0) {
      return { enabled: true, host: value.slice(0, separator), timeoutMinutes: timeout }
    }
  }

  return { enabled: true, host: value, timeoutMinutes: DEFAULT_LIVENESS_TIMEOUT }
}

export function formatProjectDomain(enabled: boolean, host: string, timeoutMinutes: number): string {
  return enabled ? `${host.trim()}:${timeoutMinutes}` : ""
}

export function isValidProjectHost(host: string): boolean {
  if (!host || host.length > 253) return false
  return host.split(".").every((label) => (
    label.length > 0
    && label.length <= 63
    && /^[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?$/.test(label)
  ))
}

export function defaultProjectCommand(name: string): string {
  const image = name.trim().toLowerCase() || "project-name"
  return `docker build -t ${image} . && exec docker run -p "$PORT":80 --rm ${image}`
}

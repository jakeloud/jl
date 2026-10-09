export interface ProjectDomain {
  enabled: boolean
  host: string
}

export function parseProjectDomain(value = ""): ProjectDomain {
  if (!value) {
    return { enabled: false, host: "" }
  }
  return { enabled: true, host: value }
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
  return `docker build -t ${image} .\ndocker run -p "$PORT":80 --rm ${image}`
}

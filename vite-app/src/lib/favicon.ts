import { parseProjectDomain } from "./projects"

export function getDomainFavicon(domain: string): Promise<string> {
  return new Promise((resolve) => {
    setTimeout(() => resolve(''), 60000)

    const { enabled, host } = parseProjectDomain(domain)
    if (!enabled || !host) {
      resolve('')
      return
    }

    const url = `https://${host}/favicon.ico`

    const image = document.createElement('img')
    image.src = url
    image.onload = () => {
      resolve(url)
    }
  })
}

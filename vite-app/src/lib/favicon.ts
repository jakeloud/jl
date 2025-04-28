export function getDomainFavicon(domain: string): Promise<string> {
  return new Promise((resolve) => {
    setTimeout(() => resolve(''), 60000)

    const url = `https://${domain}/favicon.ico`

    const image = document.createElement('img')
    image.src = url
    image.onload = () => {
      resolve(url)
    }
  })
}


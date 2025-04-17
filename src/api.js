const getLoginData = () => {
  const password = window.localStorage.getItem('pwd')
  const email = window.localStorage.getItem('email')
  return { password, email }
}

export default async function api(op, body = {}) {
  return await fetch('/api', {
    method: 'POST',
    body: JSON.stringify({ op, ...getLoginData(), ...body }),
  })
}


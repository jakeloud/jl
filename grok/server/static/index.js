let conf

const setLoginData = (pwd, email) => {
  window.localStorage.setItem('pwd', pwd)
  window.localStorage.setItem('email', email)
}
const getLoginData = () => {
  password = window.localStorage.getItem('pwd')
  email = window.localStorage.getItem('email')
  return { password, email }
}

const api = async (op, body = {}) =>
  await fetch('/api', {
    method: 'POST',
    body: JSON.stringify({op, ...getLoginData(), ...body}),
  })

const Button = (text, onClick) => {
  const button = document.createElement('button')
  button.innerText = text
  button.onclick = onClick
  return button
}

const Field = (name, type='text', initialValue=null) => {
  const field = document.createElement('div')
  const input = document.createElement('input')
  const label = document.createElement('label')
  input.id = name
  input.type = type
  input.name = name
  if (initialValue != null) {
    input.value = initialValue
  }
  label.for = name
  label.innerText = name
  field.append(label, input)
  return field
}

const Form = (onSubmit, submitText, ...fields) => {
  const form = document.createElement('form')
  form.onsubmit = onSubmit
  form.append(...fields, Button(submitText))
  return form
}

const formDataToJSON = (formData) => {
  const object = {}
  formData.forEach((value, key) => object[key] = value)
  return object
}

const handleJakeloudDomain = (e) => {
  const data = new FormData(e.target)
  e.preventDefault()
  api('setJakeloudDomainOp', formDataToJSON(data))
  location.replace(`https://${data.get('domain')}`)
}
const handleRegister = async (e) => {
  const data = new FormData(e.target)
  e.preventDefault()
  setLoginData(data.get('password'), data.get('email'))
  root.innerHTML = 'Registering...'
  await api('registerOp', formDataToJSON(data))
  getConf()
}
const handleLogin = (e) => {
  const data = new FormData(e.target)
  e.preventDefault()
  setLoginData(data.get('password'), data.get('email'))
  getConf()
}
const handleCreateApp = async (e) => {
  const data = new FormData(e.target)
  e.preventDefault()
  root.innerHTML = 'Creating app. Refresh to track progress in real time'
  await api('createAppOp', formDataToJSON(data))
  getConf()
}

add = (options = {}) => {
  root.innerHTML = ''
  const p = document.createElement('p')
  p.innerText = `Enter github repo in a format "git@github.com:<user>/<repo>.git" (as seen in ssh clone option). Example docker options: "-v /home/jakeloud:/home/jakeloud -e PASSWORD=jakeloud"`
  root.append(Form(handleCreateApp, 'create app', Field('name'), Field('domain'), Field('repo'), Field('dockerOptions'), p))
}

const handleRegisterAllowed = (prevAdditional, registerAllowed) => {
  api('setJakeloudAdditionalOp', {additional: {...prevAdditional, registerAllowed}})
}

const handleClearCache = () => {
  api('clearCacheOp')
}

const handleAttachTg = (prevAdditional) => async (e) => {
  const data = new FormData(e.target)
  e.preventDefault()
  api('setJakeloudAdditionalOp', {additional: {...prevAdditional, ...formDataToJSON(data)}})
}

const App = (app) => {
  const additional = app.additional ?? {}
  if (additional.dockerOptions) {
    app.dockerOptions = additional.dockerOptions
  }

  const wrapper = document.createElement('div')
  const info = document.createElement('pre')
  info.innerHTML = `
<a href="#${app.name}">&nwarr;</a><b>${app.name}</b> - <a href="https://${app.domain}">${app.domain}</a>
repo: ${app.repo}
owner: ${app.email}
<big>status: ${app.state}</big>`
  wrapper.append(document.createElement('hr'), info)

  if (app.name === 'jakeloud') {
    const registrationCheckbox = document.createElement('div')
    registrationCheckbox.innerHTML = `
      <input id="a" ${additional.registerAllowed === true ? 'checked' : ''} type="checkbox" onclick='handleRegisterAllowed(${JSON.stringify(additional)}, event.target.checked)'/>
      <label for="a">
      Registration allowed
      </label>`
    const telegramChatForm = Form(handleAttachTg(additional), 'attach telegram', Field('chatId', 'text', additional.chatId), Field('botToken', 'text', additional.botToken))

    wrapper.append(
      registrationCheckbox,
      telegramChatForm,
      `ssh-key:\n${additional.sshKey}`,
      Button('copy', () => {
        navigator.clipboard.writeText(additional.sshKey)
      }),
      Button('clear cache', handleClearCache),
    )
  } else {
    wrapper.append(Button('full reboot', () => api('createAppOp', app)))
    wrapper.append(
      Button('delete app', () => api('deleteAppOp', app)),
    )
  }
  return wrapper
}

let Header

const SettingsTab = () => {
  const jakeloudApp = conf.apps.find(app => app.name === 'jakeloud')
  
  root.innerHTML = ''
  root.append(
    Header(),
    Button('logout', setLoginData.bind(null, [null, null])),
    App(jakeloudApp),
  )
}

const AppsTab = () => {
  root.innerHTML = ''
  const apps = conf.apps.filter(app => {
    const hash = window.location.hash
    const isDetailedInfo = hash !== '' ? hash === `#${app.name}` : true
    return app.name !== 'jakeloud' && isDetailedInfo
  }).map(App)
  root.append(
    Header(),
    Button('add app', add),
    ...apps,
  )
}

Header = () => {
  const nav = document.createElement('nav')
  nav.append(
    Button('apps', AppsTab),
    Button('settings', SettingsTab),
    document.createElement('hr'),
  )
  return nav
}

const confHandler = {
  domain: () => {
    root.innerHTML = ''
    root.append(Form(handleJakeloudDomain, 'assign domain', Field('email', 'email'), Field('domain')))
  },
  login: () => {
    root.innerHTML = ''
    root.append(
      Form(handleLogin, 'login', Field('email', 'email'), Field('password', 'password')),
      Form(handleRegister, 'register', Field('email', 'email'), Field('password', 'password'))
    )
  },
  register: () => {
    root.innerHTML = ''
    root.append(Form(handleRegister, 'register', Field('email', 'email'), Field('password', 'password')))
  }
}

const getConf = async () => {
  const res = await api('getConfOp')
  conf = await res.json()
  if (conf.message) {
    confHandler[conf.message]()
  } else{
    AppsTab()
  }
}

onload=getConf()

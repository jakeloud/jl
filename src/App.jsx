import { useState, useEffect } from 'react';
import Register from './pages/register'

const setLoginData = (pwd, email) => {
  window.localStorage.setItem('pwd', pwd);
  window.localStorage.setItem('email', email);
};

const getLoginData = () => {
  const password = window.localStorage.getItem('pwd');
  const email = window.localStorage.getItem('email');
  return { password, email };
};

const api = async (op, body = {}) => {
  return await fetch('/api', {
    method: 'POST',
    body: JSON.stringify({ op, ...getLoginData(), ...body }),
  });
};

const Field = ({ name, type = 'text', initialValue = null }) => {
  return (
    <div>
      <label htmlFor={name}>{name}</label>
      <input id={name} type={type} name={name} defaultValue={initialValue} />
    </div>
  );
};

const Form = ({ onSubmit, submitText, children }) => {
  return (
    <form onSubmit={onSubmit}>
      {children}
      <button>
        {submitText}
      </button>
    </form>
  );
};

const formDataToJSON = (formData) => {
  const object = {};
  formData.forEach((value, key) => (object[key] = value));
  return object;
};

const AppComponent = ({ app }) => {
  const additional = app.additional ?? {};
  if (additional.dockerOptions) {
    app.dockerOptions = additional.dockerOptions;
  }

  const handleRegisterAllowed = (prevAdditional, registerAllowed) => {
    api('setJakeloudAdditionalOp', { additional: { ...prevAdditional, registerAllowed } });
  };

  const handleClearCache = () => {
    api('clearCacheOp');
  };

  const handleAttachTg = (prevAdditional) => async (e) => {
    e.preventDefault();
    const data = new FormData(e.target);
    api('setJakeloudAdditionalOp', { additional: { ...prevAdditional, ...formDataToJSON(data) } });
  };

  return (
    <div>
      <hr />
      <pre>
        <a href={`#${app.name}`}>↖</a>
        <b>{app.name}</b> - <a href={`https://${app.domain}`}>{app.domain}</a>
        <br />
        repo: {app.repo}
        <br />
        owner: {app.email}
        <br />
        <big>status: {app.state}</big>
      </pre>
      {app.name === 'jakeloud' ? (
        <>
          <div>
            <input
              id="a"
              type="checkbox"
              checked={additional.registerAllowed === true}
              onChange={(e) => handleRegisterAllowed(additional, e.target.checked)}
            />
            <label htmlFor="a">Registration allowed</label>
          </div>
          <Form onSubmit={handleAttachTg(additional)} submitText="attach telegram">
            <Field name="chatId" type="text" initialValue={additional.chatId} />
            <Field name="botToken" type="text" initialValue={additional.botToken} />
          </Form>
          <div>ssh-key: {additional.sshKey}</div>
          <button
            onClick={() => navigator.clipboard.writeText(additional.sshKey)}
          >
            copy
          </button>
          <button onClick={handleClearCache}>
            clear cache
          </button>
        </>
      ) : (
        <>
          <button
            onClick={() => api('createAppOp', app)}
          >
            full reboot
          </button>
          <button
            onClick={() => api('deleteAppOp', app)}
          >
            delete app
          </button>
        </>
      )}
    </div>
  );
};

const Header = ({ onAppsClick, onSettingsClick }) => {
  return (
    <nav>
      <button onClick={onAppsClick}>
        apps
      </button>
      <button onClick={onSettingsClick}>
        settings
      </button>
      <hr />
    </nav>
  );
};

const App = () => {
  const [conf, setConf] = useState(null);

  const handleRegister = async (e) => {
    e.preventDefault();
    const data = new FormData(e.target);
    setLoginData(data.get('password'), data.get('email'));
    api('registerOp', formDataToJSON(data));
    await getConf();
  };

  const handleLogin = (e) => {
    e.preventDefault();
    const data = new FormData(e.target);
    setLoginData(data.get('password'), data.get('email'));
    getConf();
  };

  const handleCreateApp = async (e) => {
    e.preventDefault();
    const data = new FormData(e.target);
    api('createAppOp', formDataToJSON(data));
    await getConf();
  };

  const add = () => {
    setConf({
      ...conf,
      message: 'add',
    });
  };

  const getConf = async () => {
    const res = await api('getConfOp');
    const data = await res.json();
    setConf(data);
  };

  useEffect(() => {
    getConf();
  }, []);

  const renderAddForm = () => (
    <>
      <p>
        Enter github repo in a format "git@github.com:&lt;user&rt;/&lt;repo&rt;.git" (as seen in ssh clone option). Example docker options: "-v /home/jakeloud:/home/jakeloud -e PASSWORD=jakeloud"
      </p>
      <Form onSubmit={handleCreateApp} submitText="create app">
        <Field name="name" />
        <Field name="domain" />
        <Field name="repo" />
        <Field name="dockerOptions" />
      </Form>
    </>
  );

  const renderSettingsTab = () => {
    const jakeloudApp = conf.apps.find((app) => app.name === 'jakeloud');
    return (
      <>
        <Header onAppsClick={() => setConf({ ...conf, message: null })} onSettingsClick={() => {}} />
        <button onClick={() => setLoginData(null, null)}>
          logout
        </button>
        <AppComponent app={jakeloudApp} />
      </>
    );
  };

  const renderAppsTab = () => {
    const apps = conf.apps
      .filter((app) => {
        const hash = window.location.hash;
        const isDetailedInfo = hash !== '' ? hash === `#${app.name}` : true;
        return app.name !== 'jakeloud' && isDetailedInfo;
      })
      .map((app) => <AppComponent key={app.name} app={app} />);
    return (
      <>
        <Header onAppsClick={() => {}} onSettingsClick={() => setConf({ ...conf, message: 'settings' })} />
        <button onClick={add}>
          add app
        </button>
        {apps}
      </>
    );
  };

  const renderContent = () => {
    if (!conf) return null;

    switch (conf.message) {
      case 'login':
        return (
          <>
            <Form onSubmit={handleLogin} submitText="login">
              <Field name="email" type="email" />
              <Field name="password" type="password" />
            </Form>
            <Form onSubmit={handleRegister} submitText="register">
              <Field name="email" type="email" />
              <Field name="password" type="password" />
            </Form>
          </>
        );
      case 'register':
        return <Register getConf={getConf}/>
      case 'add':
        return renderAddForm();
      case 'settings':
        return renderSettingsTab();
      default:
        return renderAppsTab();
    }
  };

  return renderContent()
};

export default App;

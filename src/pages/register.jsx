import api from '../api'
import styles from './register.module.css'

const formDataToJSON = (formData) => {
  const object = {}
  formData.forEach((value, key) => (object[key] = value))
  return object
}
const setLoginData = (pwd, email) => {
  window.localStorage.setItem('pwd', pwd);
  window.localStorage.setItem('email', email);
};

export default function Register({ getConf }) {
  const handleRegister = async (e) => {
    e.preventDefault();
    const data = new FormData(e.target);
    setLoginData(data.get('password'), data.get('email'));
    api('registerOp', formDataToJSON(data));
    await getConf();
  };

  return (
    <div className={styles.page}>
      <form
        onSubmit={handleRegister}
        className={styles.card}
      >
        <div className={styles.cardheader}>
          <h1 className={styles.h}>
            Registration
          </h1>
          <p className={styles.p}>
            Create an account to get started
          </p>
        </div>
        <div className={styles.fieldgroup}>
          <div>
            <label
              className={styles.label}
              htmlFor="email"
            >
              Email
            </label>
            <input
              className={styles.input}
              id="email"
              type="email"
              name="email"
              placeholder="Enter your email"
            />
          </div>
          <div>
            <label
              className={styles.label}
              htmlFor="password"
            >
              Password
            </label>
            <input
              className={styles.input}
              id="password"
              type="password"
              name="password"
              placeholder="Create a password"
            />
          </div>
          <button className={styles.button}>
            register
          </button>
        </div>
      </form>
    </div>
  )
}

//import { Home, Settings } from "lucide-react"
import styles from './dashboard.module.css'

export default function Dashboard({apps}) {
  console.log(apps)
  return (
    <div className={styles.page}>
      {/* Header */}
      <header className={styles.header}>
        <div className={styles.headergroup}>
          <h1 className={styles.headertitle}>Jakeloud</h1>
          <button className={styles.headerbutton}>
            {/*<Home className="h-4 w-4" />*/}
            <span>home</span>
          </button>
        </div>

        <div className={styles.headergroup}>
          <button className={styles.headerbutton}>
            {/*<Settings className="h-4 w-4" />*/}
            <span>settings</span>
          </button>
          {/*
          <Avatar className="h-10 w-10 border border-border/50">
            <AvatarFallback className="bg-muted text-muted-foreground">U</AvatarFallback>
          </Avatar>
          */}
        </div>
      </header>

      {/* Search and Filter */}
      <main className={styles.main}>
        <div className={styles.searchgroup}>
          <input placeholder="search" className={styles.search} />
          <select
            defaultValue="Filter by: last-updated"
            className={styles.filter}
          >
            <option value="Filter by: last-updated">filter by: last updated</option>
            <option value="Filter by: name">filter by: name</option>
            <option value="Filter by: status">filter by: status</option>
          </select>
        </div>

        {/* App Cards */}
        <div className={styles.grid}>
          {[1, 2, 3].map((item) => (
            <div key={item} className={styles.card}>
              <div className={styles.cardwrapper}>
                <h2 className={styles.title}>App - Website</h2>

                <div className={styles.cardcontent}>
                  <p className={styles.appstatus}>
                    Status:{' '}
                    <span className={styles.appstatustext}>
                      Up 32 hours
                    </span>
                  </p>

                  <p className={styles.appdate}>
                    updated: Mon 12 Apr 2025
                  </p>
                </div>
              </div>
            </div>
          ))}
        </div>
      </main>
    </div>
  )
}

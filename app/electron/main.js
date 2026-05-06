const { app, BrowserWindow } = require('electron')
const { spawn } = require('child_process')
const path = require('path')
const http = require('http')

const GO_PORT = 8765

let goServer
let win

function startGoServer() {
  const binName = process.platform === 'win32' ? 'server.exe' : 'server'
  const binPath = app.isPackaged
    ? path.join(process.resourcesPath, binName)
    : path.join(__dirname, '..', 'bin', binName)

  const dictPath = app.isPackaged
    ? path.join(app.getPath('userData'), 'dict.json')
    : path.join(__dirname, '..', '..', 'dict.json')

  goServer = spawn(binPath, ['-port', String(GO_PORT), '-dict', dictPath], {
    stdio: ['ignore', 'pipe', 'pipe'],
  })
  goServer.stdout.on('data', d => process.stdout.write('[go] ' + d))
  goServer.stderr.on('data', d => process.stderr.write('[go] ' + d))
  goServer.on('error', err => console.error('failed to start go server:', err.message))
}

function waitForServer(retries = 40) {
  return new Promise((resolve, reject) => {
    const attempt = n => {
      http
        .get(`http://localhost:${GO_PORT}/health`, () => resolve())
        .on('error', () => {
          if (n <= 0) return reject(new Error('Go server did not start in time'))
          setTimeout(() => attempt(n - 1), 250)
        })
    }
    attempt(retries)
  })
}

async function createWindow() {
  win = new BrowserWindow({
    width: 960,
    height: 680,
    titleBarStyle: 'hiddenInset',
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true,
    },
  })

  if (app.isPackaged) {
    win.loadFile(path.join(__dirname, '..', 'dist', 'index.html'))
  } else {
    win.loadURL('http://localhost:5173')
  }
}

app.whenReady().then(async () => {
  startGoServer()
  try {
    await waitForServer()
  } catch (e) {
    console.error(e.message)
  }
  createWindow()
})

app.on('window-all-closed', () => {
  if (goServer) goServer.kill()
  if (process.platform !== 'darwin') app.quit()
})

app.on('before-quit', () => {
  if (goServer) goServer.kill()
})

app.on('activate', () => {
  if (BrowserWindow.getAllWindows().length === 0) createWindow()
})

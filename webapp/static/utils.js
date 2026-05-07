function randomStr(len = 6) {
  return Math.random().toString(32).slice(-len)
}

function $(selector) {
  return document.querySelector(selector)
}

function $$(selector) {
  return document.querySelectorAll(selector)
}

function post(url, data) {
  return fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data)
  }).then(r => r.json())
}

function get(url) {
  return fetch(url).then(r => r.json())
}

function apiRun(cmd, dir, type = '', name = '', packageName = '', preCommand = '', skipPreCommand = false, sdkPath = '') {
  return post('/api/run', { cmd, dir, type, name, packageName, preCommand, skipPreCommand, sdkPath })
}

function apiKill(pid) {
  return post('/api/kill', { pid })
}

function apiListDir(dir) {
  return get('/api/listDir?dir=' + encodeURIComponent(dir))
}

function apiBrowse(dir) {
  return get('/api/browse?dir=' + encodeURIComponent(dir))
}

function apiGetBatContent(file) {
  return get('/api/getBatContent?file=' + encodeURIComponent(file))
}

function apiExists(path) {
  return get('/api/exists?path=' + encodeURIComponent(path))
}

function apiWatchProcess(name) {
  return post('/api/watchProcess', { name })
}

function apiGetLog(file) {
  return get('/api/getLog?file=' + encodeURIComponent(file))
}

function apiAnalyzeProject(dir, packageName) {
  let url = '/api/analyzeProject?dir=' + encodeURIComponent(dir)
  if (packageName) {
    url += '&packageName=' + encodeURIComponent(packageName)
  }
  return get(url)
}

function apiFindJar(dir) {
  return get('/api/findJar?dir=' + encodeURIComponent(dir))
}

function apiSelectDir() {
  return get('/api/selectDir')
}

function saveData(key, data) {
  localStorage.setItem(key, JSON.stringify(data))
}

function loadData(key) {
  const v = localStorage.getItem(key)
  return v ? JSON.parse(v) : null
}

function exportData(key, filename) {
  const data = loadData(key)
  if (!data) return
  const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

function importData(key, file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = e => {
      try {
        const data = JSON.parse(e.target.result)
        saveData(key, data)
        resolve(data)
      } catch (err) {
        reject(err)
      }
    }
    reader.onerror = reject
    reader.readAsText(file)
  })
}

function showToast(msg, type = 'info') {
  const toast = document.createElement('div')
  toast.className = 'toast toast-' + type
  toast.textContent = msg
  document.body.appendChild(toast)
  setTimeout(() => toast.classList.add('show'), 10)
  setTimeout(() => {
    toast.classList.remove('show')
    setTimeout(() => toast.remove(), 300)
  }, 3000)
}

function showConfirm(msg) {
  return new Promise(resolve => {
    const div = document.createElement('div')
    div.className = 'modal-mask'
    div.innerHTML = '<div class="modal-content"><div class="modal-body">' + msg + '</div><div class="modal-footer"><button class="btn-cancel">取消</button><button class="btn-confirm primary">确定</button></div></div>'
    document.body.appendChild(div)
    div.querySelector('.btn-confirm').onclick = () => { div.remove(); resolve(true) }
    div.querySelector('.btn-cancel').onclick = () => { div.remove(); resolve(false) }
    div.onclick = e => { if (e.target === div) { div.remove(); resolve(false) } }
  })
}

function formatTime(ts) {
  if (!ts) return ''
  const d = new Date(ts)
  return d.getFullYear() + '-' + String(d.getMonth() + 1).padStart(2, '0') + '-' + String(d.getDate()).padStart(2, '0') + ' ' + String(d.getHours()).padStart(2, '0') + ':' + String(d.getMinutes()).padStart(2, '0')
}

import http from './http'

export function getPluginList() {
  return http.get('/api/plugin/list')
}

export function getPluginConfig(pluginKey) {
  return http.get('/api/plugin/config', {
    params: { pluginKey }
  })
}

export function savePluginConfig(payload) {
  return http.post('/api/plugin/config/save', payload)
}

export function installPlugin(payload) {
  return http.post('/api/plugin/install', payload)
}

export function uploadPluginZip(file) {
  const form = new FormData()
  form.append('file', file)
  return http.post('/api/plugin/install/upload', form, {
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
}

export function upgradePlugin(payload) {
  return http.post('/api/plugin/upgrade', payload)
}

export function enablePlugin(payload) {
  return http.post('/api/plugin/enable', payload)
}

export function disablePlugin(payload) {
  return http.post('/api/plugin/disable', payload)
}

export function uninstallPlugin(payload) {
  return http.post('/api/plugin/uninstall', payload)
}

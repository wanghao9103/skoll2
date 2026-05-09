const runtime = window.__SKOLL_VUE__ || {}
const h = runtime.h
const ref = runtime.ref
const onMounted = runtime.onMounted

export default {
  name: 'SampleGoPluginPage',
  setup() {
    if (!h || !ref) {
      return () => null
    }

    const loading = ref(true)
    const result = ref('')
    const errText = ref('')

    async function load() {
      loading.value = true
      errText.value = ''
      try {
        const token = localStorage.getItem('token') || ''
        const res = await fetch('/api/plugin/sample-goplugin/ping', {
          headers: token ? { Authorization: `Bearer ${token}` } : {}
        })
        const body = await res.json()
        if (!res.ok) {
          throw new Error(body.message || `request failed: ${res.status}`)
        }
        result.value = JSON.stringify(body.data || body, null, 2)
      } catch (err) {
        errText.value = err.message || 'request failed'
      } finally {
        loading.value = false
      }
    }

    if (onMounted) {
      onMounted(load)
    } else {
      load()
    }

    return () => h('div', { style: 'padding:20px' }, [
      h('h2', 'Sample GoPlugin 通道示例插件'),
      h('p', '该插件用于 Linux 下 go plugin 通道验证。'),
      loading.value ? h('p', '加载中...') : null,
      errText.value ? h('p', { style: 'color:#d04040' }, errText.value) : null,
      result.value ? h('pre', { style: 'background:#f7f7f7;padding:12px;border-radius:8px' }, result.value) : null
    ])
  }
}

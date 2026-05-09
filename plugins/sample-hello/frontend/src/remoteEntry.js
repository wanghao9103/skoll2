const runtime = window.__SKOLL_VUE__ || {}
const h = runtime.h
const ref = runtime.ref
const onMounted = runtime.onMounted

export default {
  name: 'SampleHelloPlugin',
  setup() {
    if (!h || !ref) {
      return () => null
    }

    const loading = ref(true)
    const pluginConfig = ref([])
    const records = ref([])
    const title = ref('')
    const content = ref('')
    const saving = ref(false)
    const configError = ref('')
    const recordsError = ref('')

    async function loadPluginConfig() {
      loading.value = true
      configError.value = ''
      recordsError.value = ''

      try {
        const token = localStorage.getItem('token') || ''
        const res = await fetch('/api/plugin/config?pluginKey=sample-hello', {
          headers: token ? { Authorization: `Bearer ${token}` } : {}
        })

        if (!res.ok) {
          throw new Error(`request failed: ${res.status}`)
        }

        const body = await res.json()
        pluginConfig.value = body.data || []
      } catch (err) {
        configError.value = err.message || 'load config failed'
      }

      try {
        const token = localStorage.getItem('token') || ''
        const rows = await fetch('/api/plugin/sample-hello/records', {
          headers: token ? { Authorization: `Bearer ${token}` } : {}
        })
        if (!rows.ok) {
          throw new Error(`records request failed: ${rows.status}`)
        }
        const rowBody = await rows.json()
        records.value = rowBody.data || []
      } catch (err) {
        recordsError.value = err.message || 'load records failed'
      } finally {
        loading.value = false
      }
    }

    async function addRecord() {
      if (!title.value.trim()) {
        recordsError.value = '标题不能为空'
        return
      }

      saving.value = true
      recordsError.value = ''
      try {
        const token = localStorage.getItem('token') || ''
        const res = await fetch('/api/plugin/sample-hello/records', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            ...(token ? { Authorization: `Bearer ${token}` } : {})
          },
          body: JSON.stringify({
            title: title.value,
            content: content.value
          })
        })
        if (!res.ok) {
          throw new Error(`create failed: ${res.status}`)
        }

        title.value = ''
        content.value = ''
        await loadPluginConfig()
      } catch (err) {
        recordsError.value = err.message || 'create failed'
      } finally {
        saving.value = false
      }
    }

    async function removeRecord(id) {
      saving.value = true
      recordsError.value = ''
      try {
        const token = localStorage.getItem('token') || ''
        const res = await fetch(`/api/plugin/sample-hello/records/${id}`, {
          method: 'DELETE',
          headers: token ? { Authorization: `Bearer ${token}` } : {}
        })
        if (!res.ok) {
          throw new Error(`delete failed: ${res.status}`)
        }
        await loadPluginConfig()
      } catch (err) {
        recordsError.value = err.message || 'delete failed'
      } finally {
        saving.value = false
      }
    }

    if (onMounted) {
      onMounted(loadPluginConfig)
    } else {
      loadPluginConfig()
    }

    return () => h('div', { style: cardStyle }, [
      h('h2', { style: titleStyle }, 'Sample Hello 插件'),
      h('p', { style: subStyle }, '这是一个远程插件页面，已接入后端 CRUD 和插件配置读取。'),
      loading.value
        ? h('p', { style: stateStyle }, '正在加载插件配置...')
        : h('div', [
              h('h3', { style: sectionStyle }, '插件配置'),
              configError.value
                ? h('p', { style: errorStyle }, `配置加载失败: ${configError.value}`)
                : null,
              pluginConfig.value.length === 0
                ? h('p', { style: stateStyle }, '暂无配置，可在插件管理 -> 配置 中添加')
                : h('ul', { style: listStyle }, pluginConfig.value.map((item) =>
                    h('li', { style: itemStyle }, `${item.key}: ${item.isSecret ? '******' : item.value}`)
                  )),
              h('h3', { style: sectionStyle }, '示例记录'),
              recordsError.value
                ? h('p', { style: errorStyle }, `记录接口异常: ${recordsError.value}`)
                : null,
              h('div', { style: formStyle }, [
                h('input', {
                  value: title.value,
                  placeholder: '标题',
                  style: inputStyle,
                  onInput: (e) => {
                    title.value = e.target.value
                  }
                }),
                h('input', {
                  value: content.value,
                  placeholder: '内容',
                  style: inputStyle,
                  onInput: (e) => {
                    content.value = e.target.value
                  }
                }),
                h('button', {
                  style: buttonStyle,
                  disabled: saving.value,
                  onClick: addRecord
                }, saving.value ? '处理中...' : '新增')
              ]),
              records.value.length === 0
                ? h('p', { style: stateStyle }, '暂无记录')
                : h('ul', { style: listStyle }, records.value.map((row) =>
                    h('li', { style: recordRowStyle }, [
                      h('span', `${row.title} - ${row.content || ''}`),
                      h('button', {
                        style: linkButtonStyle,
                        disabled: saving.value,
                        onClick: () => removeRecord(row.id)
                      }, '删除')
                    ])
                  ))
            ])
    ])
  }
}

export const RecordsPage = {
  name: 'SampleHelloRecordsPage',
  setup() {
    if (!h || !ref) {
      return () => null
    }

    const loading = ref(true)
    const records = ref([])
    const title = ref('')
    const content = ref('')
    const saving = ref(false)
    const recordsError = ref('')

    async function loadRecords() {
      loading.value = true
      recordsError.value = ''
      try {
        const token = localStorage.getItem('token') || ''
        const rows = await fetch('/api/plugin/sample-hello/records', {
          headers: token ? { Authorization: `Bearer ${token}` } : {}
        })
        if (!rows.ok) {
          throw new Error(`records request failed: ${rows.status}`)
        }
        const rowBody = await rows.json()
        records.value = rowBody.data || []
      } catch (err) {
        recordsError.value = err.message || 'load records failed'
      } finally {
        loading.value = false
      }
    }

    async function addRecord() {
      if (!title.value.trim()) {
        recordsError.value = '标题不能为空'
        return
      }

      saving.value = true
      recordsError.value = ''
      try {
        const token = localStorage.getItem('token') || ''
        const res = await fetch('/api/plugin/sample-hello/records', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            ...(token ? { Authorization: `Bearer ${token}` } : {})
          },
          body: JSON.stringify({
            title: title.value,
            content: content.value
          })
        })
        if (!res.ok) {
          throw new Error(`create failed: ${res.status}`)
        }

        title.value = ''
        content.value = ''
        await loadRecords()
      } catch (err) {
        recordsError.value = err.message || 'create failed'
      } finally {
        saving.value = false
      }
    }

    async function removeRecord(id) {
      saving.value = true
      recordsError.value = ''
      try {
        const token = localStorage.getItem('token') || ''
        const res = await fetch(`/api/plugin/sample-hello/records/${id}`, {
          method: 'DELETE',
          headers: token ? { Authorization: `Bearer ${token}` } : {}
        })
        if (!res.ok) {
          throw new Error(`delete failed: ${res.status}`)
        }
        await loadRecords()
      } catch (err) {
        recordsError.value = err.message || 'delete failed'
      } finally {
        saving.value = false
      }
    }

    if (onMounted) {
      onMounted(loadRecords)
    } else {
      loadRecords()
    }

    return () => h('div', { style: cardStyle }, [
      h('h2', { style: titleStyle }, 'Sample Hello 记录页'),
      h('p', { style: subStyle }, '这是插件内新增页面，通过 remoteModule 路由加载。'),
      loading.value
        ? h('p', { style: stateStyle }, '正在加载记录...')
        : h('div', [
              recordsError.value
                ? h('p', { style: errorStyle }, `记录接口异常: ${recordsError.value}`)
                : null,
              h('div', { style: formStyle }, [
                h('input', {
                  value: title.value,
                  placeholder: '标题',
                  style: inputStyle,
                  onInput: (e) => {
                    title.value = e.target.value
                  }
                }),
                h('input', {
                  value: content.value,
                  placeholder: '内容',
                  style: inputStyle,
                  onInput: (e) => {
                    content.value = e.target.value
                  }
                }),
                h('button', {
                  style: buttonStyle,
                  disabled: saving.value,
                  onClick: addRecord
                }, saving.value ? '处理中...' : '新增')
              ]),
              records.value.length === 0
                ? h('p', { style: stateStyle }, '暂无记录')
                : h('ul', { style: listStyle }, records.value.map((row) =>
                    h('li', { style: recordRowStyle }, [
                      h('span', `${row.title} - ${row.content || ''}`),
                      h('button', {
                        style: linkButtonStyle,
                        disabled: saving.value,
                        onClick: () => removeRecord(row.id)
                      }, '删除')
                    ])
                  ))
            ])
    ])
  }
}

const cardStyle = 'padding:20px;border-radius:12px;background:linear-gradient(135deg,#f8fcff,#eef6ff);border:1px solid #d7e6ff'
const titleStyle = 'margin:0 0 8px 0;color:#0f3e75'
const subStyle = 'margin:0 0 14px 0;color:#4f6785'
const sectionStyle = 'margin:8px 0;color:#1e4f89;font-size:16px'
const listStyle = 'margin:0;padding-left:18px;color:#2a4f76;line-height:1.8'
const itemStyle = 'margin:0'
const stateStyle = 'margin:0;color:#607d9c'
const errorStyle = 'margin:0;color:#d04040'
const formStyle = 'display:flex;gap:8px;margin-bottom:12px;flex-wrap:wrap'
const inputStyle = 'padding:8px 10px;border:1px solid #b9d2ec;border-radius:8px;min-width:180px'
const buttonStyle = 'padding:8px 14px;border:0;border-radius:8px;background:#2f74d7;color:#fff;cursor:pointer'
const recordRowStyle = 'display:flex;justify-content:space-between;gap:10px;align-items:center'
const linkButtonStyle = 'border:0;background:transparent;color:#c0392b;cursor:pointer'

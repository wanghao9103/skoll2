function listRecords(request, runtime) {
  return runtime.db.list('sample_hello_records', 'id desc')
}

function createRecord(request, runtime) {
  const body = request.body || {}
  const title = (body.title || '').trim()
  if (!title) {
    throw new Error('title is required')
  }

  const created = runtime.db.create('sample_hello_records', {
    title,
    content: body.content || ''
  })

  runtime.cache.set('last_created_record', created, 300)
  return created
}

function updateRecord(request, runtime) {
  const id = Number((request.params || {}).id || 0)
  if (!id) {
    throw new Error('invalid id')
  }

  const body = request.body || {}
  const title = (body.title || '').trim()
  if (!title) {
    throw new Error('title is required')
  }

  return runtime.db.updateById('sample_hello_records', id, {
    title,
    content: body.content || ''
  })
}

function deleteRecord(request, runtime) {
  const id = Number((request.params || {}).id || 0)
  if (!id) {
    throw new Error('invalid id')
  }

  runtime.db.deleteById('sample_hello_records', id)
  return { id }
}

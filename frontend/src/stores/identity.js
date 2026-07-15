import { defineStore } from 'pinia'
import { ref } from 'vue'
import { identityApi } from '../api/client'

export const useIdentityStore = defineStore('identity', () => {
  const identities = ref([])
  const loading = ref(false)
  const error = ref(null)

  async function fetchIdentities() {
    loading.value = true
    error.value = null
    try {
      const data = await identityApi.list()
      identities.value = data.items || []
    } catch (e) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  async function createIdentity(name, avatarColor = '#6366f1', sortOrder = 0) {
    const data = await identityApi.create({
      name,
      avatar_color: avatarColor,
      sort_order: sortOrder,
    })
    identities.value.push(data)
    return data
  }

  async function updateIdentity(id, fields) {
    const payload = {}
    if (fields.name !== undefined) payload.name = fields.name
    if (fields.avatar_color !== undefined) payload.avatar_color = fields.avatar_color
    if (fields.sort_order !== undefined) payload.sort_order = fields.sort_order

    const data = await identityApi.update(id, payload)
    const idx = identities.value.findIndex((item) => item.id === id)
    if (idx !== -1) {
      identities.value[idx] = data
    }
    return data
  }

  async function deleteIdentity(id) {
    await identityApi.delete(id)
    identities.value = identities.value.filter((item) => item.id !== id)
  }

  async function setDefault(id) {
    await identityApi.setDefault(id)
    identities.value.forEach((item) => {
      item.is_default = item.id === id
    })
  }

  return {
    identities,
    loading,
    error,
    fetchIdentities,
    createIdentity,
    updateIdentity,
    deleteIdentity,
    setDefault,
  }
})

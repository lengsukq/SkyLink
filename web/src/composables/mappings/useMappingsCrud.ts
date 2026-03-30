import { ref } from 'vue'
import api from '../../api/client'
import type { MappingItem, MappingsListResponse } from '../../types/mappings'

export function useMappingsCrud() {
  const list = ref<MappingItem[]>([])
  const loading = ref(false)

  async function load() {
    loading.value = true
    try {
      const { data } = await api.get<MappingsListResponse>('/mappings')
      list.value = data?.list || []
    } finally {
      loading.value = false
    }
  }

  async function addMapping(form: { host: string; backend: string }) {
    await api.post('/mappings', form)
    await load()
  }

  async function updateMapping(id: number, backend: string) {
    await api.put(`/mappings/${id}`, { backend })
    await load()
  }

  async function deleteMapping(id: number) {
    await api.delete(`/mappings/${id}`)
    await load()
  }

  return {
    list,
    loading,
    load,
    addMapping,
    updateMapping,
    deleteMapping,
  }
}

import { computed, ref, watch, type Ref } from 'vue'
import type { CfDnsRecord } from '../../utils/cfRecordsCache/types'

export function useCloudflareRecordList(records: Ref<CfDnsRecord[]>) {
  const query = ref('')
  const page = ref(1)
  const pageSize = ref(20)
  const pageSizeOptions = [
    { label: '10 / 页', value: 10 },
    { label: '20 / 页', value: 20 },
    { label: '50 / 页', value: 50 },
    { label: '100 / 页', value: 100 },
  ]

  const filteredRecords = computed(() => {
    const q = (query.value || '').trim().toLowerCase()
    if (!q) return records.value
    return (records.value || []).filter((r) => {
      const name = (r?.name || '').toString().toLowerCase()
      const type = (r?.type || '').toString().toLowerCase()
      const content = (r?.content || '').toString().toLowerCase()
      return name.includes(q) || type.includes(q) || content.includes(q)
    })
  })

  const pagedRecords = computed(() => {
    const total = filteredRecords.value.length
    const ps = pageSize.value || 20
    const maxPage = Math.max(1, Math.ceil(total / ps))
    if (page.value > maxPage) page.value = 1
    const start = (page.value - 1) * ps
    return filteredRecords.value.slice(start, start + ps)
  })

  watch([query, pageSize], () => {
    page.value = 1
  })

  function onPageSizeChange(ps: number) {
    pageSize.value = ps
    page.value = 1
  }

  return {
    query,
    page,
    pageSize,
    pageSizeOptions,
    filteredRecords,
    pagedRecords,
    onPageSizeChange,
  }
}

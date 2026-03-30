import { describe, expect, it } from 'vitest'
import { nextTick, ref } from 'vue'
import { useCloudflareRecordList } from './useCloudflareRecordList'

describe('useCloudflareRecordList', () => {
  it('filters by name/type/content and resets page for query change', async () => {
    const records = ref([
      { id: '1', name: 'api.example.com', type: 'A', content: '1.1.1.1' },
      { id: '2', name: 'cdn.example.com', type: 'CNAME', content: 'target.example.com' },
    ] as any[])

    const list = useCloudflareRecordList(records as any)
    list.page.value = 2
    list.query.value = 'target'
    await nextTick()

    expect(list.page.value).toBe(1)
    expect(list.filteredRecords.value.length).toBe(1)
    expect(list.filteredRecords.value[0]?.id).toBe('2')
  })

  it('enforces valid pagination window when page size changes', () => {
    const records = ref(
      Array.from({ length: 30 }, (_, i) => ({
        id: String(i + 1),
        name: `host-${i + 1}`,
        type: 'A',
        content: `10.0.0.${i + 1}`,
      })) as any[],
    )
    const list = useCloudflareRecordList(records as any)
    list.page.value = 3
    list.onPageSizeChange(50)

    expect(list.page.value).toBe(1)
    expect(list.pagedRecords.value.length).toBe(30)
  })
})


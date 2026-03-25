import { computed, ref } from 'vue'
import { DRIVE_DEFAULT_LIST_LIMIT } from '../../constants/drive'
import type { DriveCategory, DriveEntry, DriveOrder, DriveSort } from '../../types/drive'
import { driveUserListEntries } from '../../api/driveUserClient'

export function useDriveQueryState() {
  const path = ref('')
  const q = ref('')
  const activeCategory = ref<DriveCategory>('all')
  const recursive = ref(false)
  const sort = ref<DriveSort>('name')
  const order = ref<DriveOrder>('asc')

  const loading = ref(false)
  const cursor = ref('')
  const hasMore = ref(false)
  const rows = ref<DriveEntry[]>([])

  const breadcrumb = computed(() => {
    const p = (path.value || '').trim().replaceAll('\\', '/').replace(/^\/+/, '').replace(/\/+$/, '')
    if (!p) return []
    const parts = p.split('/').filter(Boolean)
    const out: Array<{ name: string; path: string }> = []
    let acc = ''
    for (const part of parts) {
      acc = acc ? `${acc}/${part}` : part
      out.push({ name: part, path: acc })
    }
    return out
  })

  const canLoadMore = computed(() => hasMore.value && !loading.value)

  async function refresh(reset: boolean) {
    loading.value = true
    try {
      if (reset) {
        cursor.value = ''
        rows.value = []
      }

      const type = activeCategory.value === 'all' ? '' : activeCategory.value
      const res = await driveUserListEntries({
        parent_path: recursive.value ? '' : path.value,
        path_prefix: recursive.value ? path.value : '',
        recursive: recursive.value,
        type,
        q: q.value || '',
        sort: sort.value,
        order: order.value,
        cursor: cursor.value || '',
        limit: DRIVE_DEFAULT_LIST_LIMIT,
        include_dirs: !type,
      })
      const incoming = (res.list || []) as DriveEntry[]
      rows.value = reset ? incoming : rows.value.concat(incoming)
      cursor.value = String(res.next_cursor || '')
      hasMore.value = !!cursor.value
    } finally {
      loading.value = false
    }
  }

  function goTo(p: string) {
    path.value = p
    q.value = ''
    cursor.value = ''
    hasMore.value = false
    rows.value = []
  }

  return {
    // query state
    path,
    q,
    activeCategory,
    recursive,
    sort,
    order,
    // list state
    loading,
    cursor,
    hasMore,
    rows,
    // derived
    breadcrumb,
    canLoadMore,
    // actions
    refresh,
    goTo,
  }
}


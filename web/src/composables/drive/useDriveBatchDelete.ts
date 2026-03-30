import { ref, type Ref } from 'vue'

export function useDriveBatchDelete(selectedPaths: Ref<string[]>, batchDeleteRaw: (paths: string[]) => Promise<void>) {
  const batchDeleteModal = ref(false)
  const batchDeleting = ref(false)
  const pendingDeleteCount = ref(0)

  function confirmBatchDelete() {
    const n = selectedPaths.value.length
    if (!n) return
    pendingDeleteCount.value = n
    batchDeleteModal.value = true
  }

  async function runBatchDelete() {
    batchDeleting.value = true
    try {
      await batchDeleteRaw(selectedPaths.value)
      batchDeleteModal.value = false
    } finally {
      batchDeleting.value = false
    }
  }

  return {
    batchDeleteModal,
    batchDeleting,
    pendingDeleteCount,
    confirmBatchDelete,
    runBatchDelete,
  }
}


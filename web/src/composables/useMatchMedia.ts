import { onMounted, onUnmounted, ref, type Ref } from 'vue'

/** Reactive `window.matchMedia` subscription. */
export function useMatchMedia(query: string): Ref<boolean> {
  const matches = ref(typeof window !== 'undefined' ? window.matchMedia(query).matches : false)
  let mql: MediaQueryList | null = null

  function sync() {
    if (mql) matches.value = mql.matches
  }

  onMounted(() => {
    mql = window.matchMedia(query)
    matches.value = mql.matches
    mql.addEventListener('change', sync)
  })

  onUnmounted(() => {
    mql?.removeEventListener('change', sync)
  })

  return matches
}

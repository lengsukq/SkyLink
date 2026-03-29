import type { Router } from 'vue-router'

export type AdminNavItem = { path: string; label: string }

export function getAdminNavItems(router: Router, isWindows: boolean): AdminNavItem[] {
  const items: { path: string; label: string; order: number }[] = []
  for (const r of router.getRoutes()) {
    const meta = r.meta
    if (!meta?.navLabel) continue
    if (meta.requiresWindows && !isWindows) continue
    if (!r.name) continue
    const resolved = router.resolve({ name: r.name })
    items.push({
      path: resolved.path,
      label: meta.navLabel,
      order: meta.navOrder ?? 999,
    })
  }
  return items.sort((a, b) => a.order - b.order).map(({ path, label }) => ({ path, label }))
}

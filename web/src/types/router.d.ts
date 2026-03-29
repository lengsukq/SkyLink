import 'vue-router'

declare module 'vue-router' {
  interface RouteMeta {
    /** Shown in admin top navigation when set */
    navLabel?: string
    navOrder?: number
    /** Hide nav item when not on Windows */
    requiresWindows?: boolean
  }
}

export {}

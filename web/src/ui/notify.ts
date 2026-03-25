type NaiveNotifier = {
  success: (opts: { title: string; content: string; duration?: number }) => void
  error: (opts: { title: string; content: string; duration?: number }) => void
}

let notifier: NaiveNotifier | null = null

export function setNotifier(n: NaiveNotifier | null) {
  notifier = n
}

export function notifySuccess(title: string, content: string) {
  notifier?.success({
    title,
    content,
    duration: 2500,
  })
}

export function notifyError(title: string, content?: string) {
  notifier?.error({
    title,
    content: content ?? '',
    duration: 5000,
  })
}


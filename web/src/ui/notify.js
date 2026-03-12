let notifier = null

export function setNotifier(n) {
  notifier = n
}

export function notifySuccess(title, content) {
  if (!notifier) return
  notifier.success({
    title,
    content,
    duration: 2500,
  })
}

export function notifyError(title, content) {
  if (!notifier) return
  notifier.error({
    title,
    content,
    duration: 5000,
  })
}


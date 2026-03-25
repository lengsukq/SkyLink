import { notifyError, notifySuccess } from '../ui/notify'

export function useDirectoryPicker(onResolvedPath) {
  function openDirectoryPicker(inputRef) {
    const input = inputRef?.value
    if (!input) {
      notifyError('当前环境不支持', '无法打开目录选择器，请手动填写本地目录路径。')
      return
    }
    input.value = ''
    input.click()
  }

  function onDirectoryPicked(event) {
    const input = event?.target
    const files = input?.files
    if (!files || files.length === 0) return

    const first = files[0]
    const relativePath = first.webkitRelativePath || ''
    const firstSegment = relativePath.split('/')[0] || ''
    const absoluteFilePath = typeof first.path === 'string' ? first.path : ''

    if (absoluteFilePath && relativePath) {
      const normalizedAbs = absoluteFilePath.replace(/\\/g, '/')
      const normalizedRel = relativePath.replace(/\\/g, '/')
      const suffix = `/${normalizedRel}`
      if (normalizedAbs.endsWith(suffix)) {
        const base = normalizedAbs.slice(0, normalizedAbs.length - suffix.length)
        const resolvedPath = base.replace(/\//g, '\\')
        onResolvedPath(resolvedPath)
        notifySuccess('已填充目录', `已自动填充：${resolvedPath}`)
        return
      }
    }

    if (firstSegment) {
      onResolvedPath(firstSegment, { partial: true })
      notifyError('无法获取绝对路径', '当前浏览器受限，已尝试提取目录名，请手动补全本地绝对路径。')
      return
    }

    notifyError('选择失败', '未能读取目录信息，请手动填写本地目录路径。')
  }

  return {
    openDirectoryPicker,
    onDirectoryPicked,
  }
}

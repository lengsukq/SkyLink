/**
 * 规范化用户输入的本机绝对路径（用于 SMB/WebDAV/网盘根目录等）。
 * Windows 下支持 `C:\Users\name`、`C:/Users/name`、`"C:\path with space"` 等形式。
 */
export function normalizeLocalPath(input: string, isWindows: boolean): string {
  let s = input.trim().replace(/^["']|["']$/g, '')
  if (!isWindows) {
    return s
  }
  // 盘符路径：统一为正斜杠再转反斜杠，避免只支持一种写法
  if (/^[a-zA-Z]:/.test(s)) {
    s = s.replace(/\//g, '\\')
  }
  // UNC
  if (s.startsWith('\\\\')) {
    s = '\\\\' + s.slice(2).replace(/\//g, '\\')
  }
  return s
}

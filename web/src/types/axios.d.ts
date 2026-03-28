import 'axios'

declare module 'axios' {
  interface AxiosRequestConfig {
    /** 为 true 时全局错误提示（拦截器）不展示该次请求的错误 */
    silentError?: boolean
  }
}

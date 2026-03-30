package easytier

import "runtime"

// HostNotSupportedMessage 为非 Windows 主机上被禁止的写操作返回的错误文案。
const HostNotSupportedMessage = "EasyTier 守护进程、端口释放与运行时下载仅在 Windows 主机上支持；请在 Windows 上运行 SkyLink 以使用集成功能。"

// HostNotSupportedHint 用于状态类 JSON（仍返回 200）中的 hint 字段。
const HostNotSupportedHint = "当前 SkyLink 未运行在 Windows 上，集成的 EasyTier 状态与 CLI 功能不可用；请在 Windows 部署或自行在目标环境运行 EasyTier。"

// EasyTierSupportedOnHost 为 true 时表示本进程所在平台支持 SkyLink 内置的 EasyTier 管理与守护进程（仅 Windows）。
func EasyTierSupportedOnHost() bool {
	return runtime.GOOS == "windows"
}

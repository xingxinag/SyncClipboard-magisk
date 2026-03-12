package clipboard

type methodFuncRead func() (string, error)
type methodFuncWrite func(string) error

type methodRead struct {
	name string
	fn   methodFuncRead
}

type methodWrite struct {
	name string
	fn   methodFuncWrite
}

type strategy struct {
	readOrder  []methodRead
	writeOrder []methodWrite
}

func detectClipboardStrategy() strategy {
	// root-first 固定顺序：优先使用真实剪贴板方法
	s := strategy{
		readOrder: []methodRead{
			{name: "apk_helper", fn: getClipboardAPK},         // 最优：通过 APK 访问真实系统剪贴板
			{name: "shared_file", fn: getClipboardSharedFile}, // 次优：共享文件（降级方案）
			{name: "termux", fn: getClipboardTermux},          // 备选：Termux API（如果安装）
			{name: "service_call", fn: getClipboardServiceCall},
			{name: "dumpsys", fn: getClipboardDumpsys},
			{name: "database", fn: getClipboardDatabase},
			{name: "shared_memory", fn: getClipboardSharedMemory},
		},
		writeOrder: []methodWrite{
			{name: "apk_helper", fn: setClipboardAPK},         // 最优：通过 APK 访问真实系统剪贴板
			{name: "shared_file", fn: setClipboardSharedFile}, // 次优：共享文件（降级方案）
			{name: "termux", fn: setClipboardTermux},          // 备选：Termux API（如果安装）
			{name: "service_call", fn: setClipboardServiceCall},
			{name: "database", fn: setClipboardDatabase},
			{name: "shared_memory", fn: setClipboardSharedMemory},
			{name: "cmd_clipboard", fn: setClipboardCmd},
		},
	}

	return s
}

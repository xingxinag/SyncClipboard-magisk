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
	// system-first 固定顺序：优先 system_server hook，移除 helper/shared 旧路径
	s := strategy{
		readOrder: []methodRead{
			{name: "system_hook", fn: getClipboardSystemHook},
			{name: "service_call", fn: getClipboardServiceCall},
			{name: "dumpsys", fn: getClipboardDumpsys},
			{name: "database", fn: getClipboardDatabase},
			{name: "shared_memory", fn: getClipboardSharedMemory},
		},
		writeOrder: []methodWrite{
			{name: "system_hook", fn: setClipboardSystemHook},
			{name: "service_call", fn: setClipboardServiceCall},
			{name: "database", fn: setClipboardDatabase},
			{name: "shared_memory", fn: setClipboardSharedMemory},
			{name: "cmd_clipboard", fn: setClipboardCmd},
		},
	}

	return s
}

package main

import "fmt"

func main() {
	// 判断参数是否存在，如果不存在要求用户输入
	InitArguments()

	fmt.Println("bvid:", BvId)
	fmt.Println("qn:", Qn)
	ColorsPrintF("下载完成!   ", 32, true, false)
}

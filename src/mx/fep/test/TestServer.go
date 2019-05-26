package main


import (
	"mx/fep"
	"runtime"
)

/**
 测试服务器端
 */
func main(){
	runtime.GOMAXPROCS(4)
	fep.StartTcpServer(18080);
}

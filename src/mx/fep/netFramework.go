package fep

import (
	"net"
	"strconv"
	"fmt"
	"runtime/debug"
)

type ServerHandler struct {
	workerChan   chan *ServerHandler
	closeChan    chan *ServerHandler
	conn         *net.TCPConn
	buf          []byte
	readN        int
	err          error
	frameContext *FrameContext
}

func newServerHandler(conn *net.TCPConn, workerChan,closeChan chan *ServerHandler) *ServerHandler {
	var sh = new(ServerHandler)
	sh.conn = conn
	//通常报文的最大长度是2000
	sh.buf = make([]byte,2000)
	sh.frameContext = new(FrameContext)
	sh.workerChan = workerChan
	sh.closeChan = closeChan

	return sh
}

/*
 从socket阻塞读取数据,并交给worker处理,worker处理完毕后会再次启动receiveRoutine,完成一次循环
 直到在读取过程中出现错误,将ServerHandler交给closer处理,进行销毁工作
 */
func (sh *ServerHandler) receiveRoutine(){
	//一旦读取出现异常,就关闭socket
	defer sh.doRecover(true)
	//阻塞读取
	var n,err = sh.conn.Read(sh.buf)
	if err!=nil {
		sh.err = err
		//送入close routine处理
		sh.closeChan<-sh
		return
	}
	//本次从socket共写入的字节数
	sh.readN = n

	//送入worker处理,必须在receiveRoutine的最后一行,保证receiveRoutine()和handle()不会并发
	sh.workerChan<-sh
}

/*
 在worker中调用,消费这一次从socket读到的数据,并再次启动receiveRoutine
 */
func (sh *ServerHandler) handle(){
	if n := sh.readN; n>0 {
		//写入frameContext
		sh.frameContext.WriteNBytes(sh.buf,n)

	    brk0:
		for{
			//尝试解析帧结构
			frame,err:=sh.frameContext.ReadFrame3761()

			//读到异常,应该为notEnoughReadable,就退出循环
			if err!=nil{
				break brk0
			}
			//---------------------消费这个frame------------------
			if frame!=nil{
				//fmt.Printf("%+v\n",frame)
				//配合测试客户端
				sh.conn.Write([]byte{250})
			}
		}
	}

	//再次启动receiveRoutine
	go sh.receiveRoutine();
}

func (sh *ServerHandler) doRecover(isClose bool){
	if err := recover(); err != nil {
		fmt.Println("截获到panic:", err)
		debug.PrintStack()

		if isClose{
			sh.closeChan<-sh
		}
	}
}

func (sh *ServerHandler) close(){
	sh.workerChan = nil
	sh.closeChan = nil
	sh.buf = nil
	sh.frameContext = nil
	//关闭socket
	defer sh.doRecover(false)
	sh.conn.Close()
}

const (
	//worker的数量,跟netty一样,worker数是一个固定的值
	WORKER_SUM = 4
)


func StartTcpServer(port int){
	listener, err := net.Listen("tcp4", ":"+strconv.Itoa(port))
	if err != nil {
		fmt.Printf("listen error:%s", err)
		return
	}

	//create worker
	var workerChans [WORKER_SUM]chan *ServerHandler
	for i:=0;i< WORKER_SUM;i++{
		workerChans[i] = make(chan *ServerHandler,100000)
		var ch = workerChans[i]
		//启动worker
		go func(){
			for sh:= range ch{
				sh.handle()
			}
		}()
	}

	//closer
	var closeChan = make(chan *ServerHandler,100000)
	go func(){
		for sh:= range closeChan{
			sh.close()
		}
	}()


	//接受连接
	var seq = uint64(0)
	for {
		var tcpListener = listener.(*net.TCPListener)

		conn, err := tcpListener.AcceptTCP();
		if err != nil {
			fmt.Printf("accept error:%s", err)
		} else {
			conn.SetKeepAlive(true)
			conn.SetNoDelay(true)
			//增加读缓冲区可以减少"循环"次数,由8192改成2000,貌似提高了性能,貌似2000是个经验值,大了,小了,或不写性能都不行
			conn.SetReadBuffer(2000)
			//增加读缓冲区,可以减少write时阻塞的时间
			conn.SetWriteBuffer(2000)

			seq++;
			var mod = int(seq%WORKER_SUM)
			//创建handler
			var sh = newServerHandler(conn,workerChans[mod],closeChan);

			//开始阻塞读取
			go sh.receiveRoutine()
		}
	}
}

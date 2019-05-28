/**
 测试客户端
 */
package main

import (
	"sync"
	"net"
	"bufio"
	"time"
	"fmt"
	"sync/atomic"
	"mx/byteUtils"
	"runtime"
)

/**
 * 保持的连接总数
 */
var ConnNum int32 = 0

/**
 * 这一个批次以来的,响应数
 */
var ResponseNum int32 = 0

type client struct {
	conn  *net.TCPConn
}

func (c *client) close() {
	c.conn.Close();
	atomic.AddInt32(&ConnNum, -1)
}


func (c *client) write(bs []byte){
	_, err := c.conn.Write(bs)
	if err!=nil {
		c.close()
	}
}


func newClient(tcpConn *net.TCPConn) *client {
	var c *client = new(client)
	c.conn = tcpConn

	atomic.AddInt32(&ConnNum, 1)

	/**
	 * 接收采集前置的响应
	 */
	var reader = bufio.NewReader(tcpConn)
	go func() {
		for{
			response, err := byteUtils.ReadN(1, reader)
			if response != nil && response[0] == 250{
				atomic.AddInt32(&ResponseNum, 1)
			}else{
				fmt.Println("读取错误",err)
				c.close()
				goto theEnd0
			}
		}

	theEnd0:
	}()

	return c
}


/*
 connNum:连接数,在每机器/虚拟机下不超过6万个,而且需要修改系统参数
 writeNum:每个连接发送的报文数量
 */
func TestClient(connNum,writeNum int,serverAddress string){
	var remoteAddress, _ = net.ResolveTCPAddr("tcp4", serverAddress)

	//monitor routine
	go func() {
		for{
			time.Sleep(1*time.Second)
			fmt.Println("当前连接数",ConnNum,"---当前响应数",ResponseNum)
			fmt.Println()
		}
	}()

	//建立连接,直到maxNum,这个时间可能会比较长
	var clients []*client
	for num := 0; num < connNum; {
		var conn, err = net.DialTCP("tcp4", nil, remoteAddress)
		if err != nil {
			fmt.Println("连接出错：", err)
		} else {
			num++
			conn.SetNoDelay(true)
			conn.SetKeepAlive(true)
			//conn.SetReadBuffer(512)
			//conn.SetWriteBuffer(512)
			var c = newClient(conn)
			clients = append(clients,c)
		}
	}

	fmt.Println("已经建立了", connNum,"个连接")

	//作为压测,并发不停的发送writeNum次
	for i:=0;i<writeNum;i++{
		writeConcurrent(clients,2)
		time.Sleep(2*time.Millisecond)
	}


	fmt.Println("已经发送了", connNum,"个上行报文")


	//主routine强制不退出
	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}

/**
 一次向clients发送请求,直到所有客户端都发出去了再返回
 concurrent为并发因子
 */
func writeConcurrent(clients []*client, concurrentFactor int){
	var len =len(clients)
	var regionLen = len/ concurrentFactor;

	var wg sync.WaitGroup
	wg.Add(concurrentFactor)

	var start = 0;
	for i:=0;i< concurrentFactor-1;i++{
		var end = start+regionLen;
		writeSliceAsync(clients[start:start+regionLen],&wg);
		start = end;
	}
	writeSliceAsync(clients[start:len],&wg);
	wg.Wait();
}


/**
 异步的发送一个客户端集合的分片
 */
func writeSliceAsync(clients []*client,group *sync.WaitGroup){
	go func() {
		for _,c:=range clients{
			c.write(f0)
		}
		group.Done()
	}()
}

/*
  测试报文,来源于百度共享文档
  https://wenku.baidu.com/view/e373edc1fe4733687e21aaff.html
  <<376.1报文解析示例>>
 */
var datagramFromBaidu  =  "68 F6 00 F6 00 68 A8 03 61 D7 22 22 09 E2 00 00 01 00 47 57 46 4B 46 4B 47 41 32 33 00 00 36 31 38 30 20 " +
	"06 14 30 32 39 2E 31 31 32 2E 33 31 30 31 33 37 36 31 30 34 30 02 01 14 0B 00 08 59 19 16 23 00 53 16"
var f0, _ = byteUtils.HexToSlice(datagramFromBaidu);


func main(){
	runtime.GOMAXPROCS(2)
	TestClient(10000,10,"127.0.0.1:18080")
}
/**
 模仿netty的byteBuf,增加回溯功能
 并不封装reader，由应从层调用write方法来写入.
 所有的方法都不阻塞,只在routine内(actor内)使用,也不考虑routine safe的问题
 */
package fep

import (
	"errors"
	"fmt"
	"mx/byteUtils"
)

type ByteBuf struct {
	//初始化为nil(==nil),但buf有地址
	buf []byte

	//下一个要读取的字节
	readIndex int
	//下一个要写入的字节
	writeIndex int

	//可以重置的下一个要读取的字节
	markIndex int
}


var (
	NotEnoughReadable = errors.New("not enough readable bytes")
	ParamError        = errors.New("param error")
)
/**
 从byteBuf中读到指定slice，填不满slice会返回error
 方法会清空bs,然后拷贝满bs的长度为止,否则会返回异常
 */
func (bf *ByteBuf) Read(bs []byte) error {
	var len = len(bs)
	if len > 0 && bf.writeIndex-bf.readIndex >= len {
		copy(bs, bf.buf[bf.readIndex:bf.readIndex+len]);
		bf.readIndex += len
		return nil;
	}
	return NotEnoughReadable
}

func (bf *ByteBuf) readableBytes() int {
	return bf.writeIndex - bf.readIndex
}

/**
 读所有可读数据,若无数据可读,返回nil
 直接截取内部slice,效率稍高,但存在返回值被修改的危险
 */
func (bf *ByteBuf) ReadAll() []byte {
	var l = bf.writeIndex - bf.readIndex
	if l > 0 {
		var bs = bf.buf[bf.readIndex:]
		bf.readIndex = bf.writeIndex
		return bs
	}

	return nil
}

/**
 同readAll
 */
func (bf *ByteBuf) ReadNBytes(n int) ([]byte, error) {
	if n <= 0 {
		return nil, ParamError
	}
	if bf.writeIndex-bf.readIndex >= n {
		var end = bf.readIndex + n
		var bs = bf.buf[bf.readIndex:end]
		bf.readIndex = end
		return bs, nil
	}

	return nil, NotEnoughReadable
}

/**
  读一个字节,若无字节可读,返回error
 */
func (bf *ByteBuf) ReadByte() (byte, error) {
	if bf.writeIndex-bf.readIndex > 0 {
		var b = bf.buf[bf.readIndex]
		bf.readIndex++
		return b, nil
	}

	return 0, NotEnoughReadable
}

/**
 将指定slice写入buf
 */
func (bf *ByteBuf) Write(bs []byte) {
	bf.buf = append(bf.buf, bs...)
	bf.writeIndex += len(bs)
}

/**
 将指定slice的前n个字节写入buf,并返回写入长度
 */
func (bf *ByteBuf) WriteNBytes(bs []byte,n int) int{
	var l = len(bs)
	if n > l {
		n = l
	}

	bf.Write(bs[:n])

	return n
}


/**
 在读之前标记,并回退n个字节
 */
func (bf *ByteBuf) MarkReadIndex(n int) {
	bf.markIndex = bf.readIndex - n;
	if bf.markIndex < 0 {
		bf.markIndex = 0
	}
}

/**
 回到mark标记的后n个字节
 */
func (bf *ByteBuf) ResetReadIndex(n int) {
	if bf.markIndex >= 0 {
		bf.readIndex = bf.markIndex + n
		if bf.readIndex > bf.writeIndex {
			bf.readIndex = bf.writeIndex
		}
		bf.markIndex = 0
	}
}

/**
 清空已读数据,减少内存占用
 */
func (bf *ByteBuf) Discard() {
	if bf.readIndex > 0 {
		bf.buf = bf.buf[bf.readIndex:]
		bf.writeIndex = bf.writeIndex - bf.readIndex
		bf.readIndex = 0
		bf.markIndex = 0
	}
}

/**
 彻底清空，以待重用
 */
func (bf *ByteBuf) Clear() {
	//地址不变，减少内存消耗
	bf.buf = bf.buf[0:0]
	bf.readIndex = 0
	bf.writeIndex = 0
	bf.markIndex = 0
}

func (bf *ByteBuf) debug() {
	fmt.Printf("readIndex:%d,writeIndex:%d,markIndex:%d\n",bf.readIndex,bf.writeIndex,bf.markIndex)
	fmt.Println("bs:",byteUtils.SliceToHex(bf.buf))
}





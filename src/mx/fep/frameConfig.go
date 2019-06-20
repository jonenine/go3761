package fep

import (
	"bytes"
	"fmt"
	"errors"
)

type Frame3761 struct {
	//当前item的slice
	itemSlice []byte
	length    int
	control   uint8
	a1        uint8
	a2        uint16
	a3        uint8
	usrData   []byte
	cs        uint8
}

type ItemConfig struct {
	name      string
	//凡是需要作为计算索引使用的field,一定要定义为int,不要为了节省点空间定义为其他类型
	length    int
	getLength func(f *Frame3761) int
	content   []byte
	validate  func(f *Frame3761) bool
	process   func(f *Frame3761)
	isCs      bool
}

func (ic *ItemConfig) Name() string{
	return ic.name
}

/*---------------------------------------376.1规约帧定义----------------------------------------*/

var Frame3761Config = [...]ItemConfig{
	//帧头,帧头比较特殊,不能使用任何方法.帧头不依赖于前面的解析结果(帧头位置在最前面),也不向frame结果输出任何值(帧头一般都固定值,也没意义)
	{"head",   1, nil,  []byte{0x68},
	 nil, nil,
	 false},

	//长度域
	{"length", 4, nil,  nil,
	 func(f *Frame3761) bool {
		var len1 = f.itemSlice[0:2]
		var len2 = f.itemSlice[2:4]

		if bytes.Compare(len1, len2) == 0 {
			f.length = (int(len1[0]) + int(len1[1])*256) >> 2
			return true;
		}
		return false;
	  },
	 nil,
	 false},

	//68
	{"next68", 1, nil,	 []byte{0x68},
	 nil,
	 nil,
	 false},

	//控制域
	{"control",1, nil,  nil,
	 nil,
	 func(f *Frame3761) {
		 f.control = uint8(f.itemSlice[0])
	  },
	 true},

	//地址域a1
	{"a1",    2,  nil,  nil,
	 nil,
	 func(f *Frame3761) {
		f.a1 = uint8(f.itemSlice[0])*10 + uint8(f.itemSlice[1])
     },
     true},

	//地址域a2
	{"a2",    2,  nil,  nil,
	 nil,
	 func(f *Frame3761) {
		f.a2 = uint16(f.itemSlice[0])*256 + uint16(f.itemSlice[1])
	 },
	 true},

	//地址域a3
	{"a3",    1,  nil,  nil,
	 nil,
	 func(f *Frame3761) {
		f.a3 = uint8(f.itemSlice[0])
	  },
	 true},

	 //数据负载
	{"userData", 0,
	 func(f *Frame3761) int {
		return f.length - 6
	  },
	 nil,
	 nil,
	 func(f *Frame3761) {
		 f.usrData = append(f.usrData,f.itemSlice...)
	  },
	 true},

	 //校验和
	{"cs",  1, 	 nil, 	 nil,
	 func(f *Frame3761) bool {
		var cs1 = uint8(f.itemSlice[0])
		return cs1 == f.cs
	 },
	 nil,
	 false},

	 //帧尾
	{"tail",1,	 nil,	 []byte{0x16},
	 nil,
	 nil,
	 false},
}

/**
  376.1规约的校验和
 */
func computeCs(cs uint8, bs ...byte) uint8 {
	for _, b := range bs {
		cs += uint8(b)
	}
	return cs
}

/*-------------------------------解析一个数据项-----------------------------------*/

/**
 继承byteBuf
 */
type FrameContext struct {
	ByteBuf
	//当前解析到帧结构的哪个索引项
	frameItemIndex int
	//当前解析frame
	parseFrame     *Frame3761
}

var ItemConfigInvalidateError = errors.New("数据项解析错误")

/*
 从bf中读一个数据项,如果返回itemConfigInvalidateError,就书名解析错误
 */
func (con *FrameContext)ReadItemConfig(ic *ItemConfig,f *Frame3761) error{
	//取得当前项长度
	var len = ic.length
	if len <= 0{
		len = ic.getLength(f)
	}
	if len <= 0 {
		fmt.Printf("数据项"+ic.name+"小于等于0\n",len)
		return ItemConfigInvalidateError
	}

	//取得item对应的slice
	bs,err:= con.ReadNBytes(len)
	if err!=nil{
		return err
	}

	//赋值给frame的itemSlice
	f.itemSlice = bs

	//验证
	if ic.content!=nil{
		if bytes.Compare(bs,ic.content)!=0{
			//fmt.Printf("数据项"+ic.name +"验证失败,应为"+test3761.SliceToHex(ic.content)+"实际为:"+test3761.SliceToHex(bs)+"\n")
			return ItemConfigInvalidateError
		}
	}else if ic.validate!=nil{
		if !ic.validate(f) {
			//fmt.Printf("数据项"+ic.name +"validate方法验证失败\n");
			return ItemConfigInvalidateError
		}
	}
	//处理
	if ic.process!=nil {
		ic.process(f);
	}

	//校验和
	if ic.isCs {
		f.cs = computeCs(f.cs,bs...);
	}


	return nil
}

/*-------------------------------解析帧结构-----------------------------------*/

/*
 一个连接对应一个唯一的frameContext
 应该一直调用ReadFrame3761方法,直到error不为nil(应该为notEnoughReadable)
 */
func (con *FrameContext) ReadFrame3761() (*Frame3761,error){
	//保证context中frame属性不是nil
	if con.parseFrame == nil {
		con.parseFrame = new(Frame3761)
	}

	//得到当前解析的数据项配置
	var ic = &Frame3761Config[con.frameItemIndex];
	//尝试解析
	var err = con.ReadItemConfig(ic,con.parseFrame);

	if err!=nil {
		//如果是验证错误
		if err == ItemConfigInvalidateError {
			//如果在非帧头的位置出错,就恢复到曾经认为的帧头的下一个字节重新解析
			if con.frameItemIndex>0 {
				con.ResetReadIndex(0)
			}
			//错误的部分数据清空
			con.Discard();
			//将数据项指到0,下一次继续从帧头进行解析
			con.frameItemIndex = 0
			//清空校验和
			con.parseFrame.cs = 0
		}else{//非notEnoughReadable,出现了意料之外的错误
			if err!= NotEnoughReadable {
				fmt.Printf("出现了意外的错误:%s\n",err)
				//基于byteBuf不应该出现其他异常,又没有底层封装connection的reader
				panic("出现意外的异常")
			}
			return nil,err
		}
	}else{//如果没有出现错误
		//解析数据项索引+1
		con.frameItemIndex++

		if con.frameItemIndex == 1{//解析完帧头
			//记录帧头的下一个位置,如果后续解析出现错误,就恢复到这个位置
			con.MarkReadIndex(0)
		}else if con.frameItemIndex == len(Frame3761Config){ //解析完帧尾
			var frame = con.parseFrame
			//清空frame
			con.parseFrame = nil
			//清空已经解析数据,减少内存占用
			con.Discard();
			//将数据项指到0
			con.frameItemIndex = 0

			return frame,nil
		}
	}

	return nil,nil
}


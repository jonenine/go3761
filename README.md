# go3761

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;实现了稳定而高效的网络通讯层和灵活的规约配置层，可作为物联网行业部署在云端的采集前置程序的原型。

&nbsp;&nbsp;&nbsp;&nbsp;网络层架构
-----------------------------------------------

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;网络层分为io routine和worker routine，有点类似于netty的网络模型。当accept连接之后就启动一个routine来接收数据，一旦接收到数据，routine退出，同时将handler连同数据（事件）通过channel发给worker进行处理，在一个固定数量的worker group（池）内进行解帧和其他业务处理。处理之后再启动一个新的routine来接收数据。
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;以前采取io routine同时接收数据和处理数据的做法，虽然性能也不错。发现一段时间后golang的后台线程疯狂增加，而采用这种类似于reactor的模式不但提高了性能，也克服了后台线程飞涨的问题。

&nbsp;&nbsp;&nbsp;&nbsp;性能测试
-------------------------------------
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;性能测试结果还是不错的:
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;在windows上似乎很难突破c10k的限制，并发数超过1万是没问题（可以并发连接到到很大），但并发超过1万后性能急剧下降，比如并发1万1千和并发一万的性能差距相当大。应该是操作系统的问题引起的，我用的server2008R2 enterprise。

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;在linux上性能就好了很多，以下是测试数据

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;**硬件配置**
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;联想低配服务器
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;8 核心Intel(R) Xeon(R) CPU E3-1230 v6 @ 3.50GHz
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;内存32G

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;**软件环境**
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;Redhat6.8
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;golang1.12.5

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;**在单节点上轻松实现c50k**,
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;5万客户端不停发送测试报文，服务端每秒40万次解帧,
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;从proc/xxxx/status看，线程数19个，内存占用(vmRss)保持在420M左右
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;资源控制的相当好，程序也很稳定。

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;因为测试客户端和server在同一台机器上，cpu已经基本压满。单台服务器的测试客户端数量（端口数）也不可能突破6万，条件所限没有继续测试下去。不过，就测试程序的轻松表现来看。在生产环境下应该可以轻松实现c100k。
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;本程序只是实现了376.1规约的解帧，实现了网路层框架而已。而实际的采集程序业务要复杂的多。本程序可以作为golang前置机的原型程序。相信在这个原型基础上可以作出高性能的成熟稳定的golang前置采集程序。

&nbsp;&nbsp;&nbsp;&nbsp;关键数据类型
------------------------------

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;ByteBuf
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;在按照某种规约或约定解析tcp的字节流的时候，往往会出现错误的模式匹配。比如你规定了业务帧的帧头为某些按顺序出现的byte常量，但这个常量组合也极有可能出现在非帧头的部分。当发现匹配失败了之后，通常需要向前回溯。
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;而java netty的ByteBuf就拥有markReaderIndex，resetReaderIndex方法，可以在解析的时候做标记，发现匹配错误之后可以回到标记的位置。


&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;ItemConfig和Frame3761Config
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;ItemConfig是一种通用的帧结构解析配置方式。将规约帧结构的每一个数据段都用ItemConfig来进行配置，每个ItemConfig定义一个帧结构段（其实就是各种数据域，如长度域、控制域）的长度，验证，输出，是否计算校验和等几个部分。Frame3761Config数组再将所有的帧结构段组合起来成为完整的帧结构。
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;解析方法ReadFrame3761，采取非常谨慎的容错策略，当解析到某个规约项配置的validate方法失败后，程序会回溯到当前认为的帧头的下一个字节去重新匹配帧头。这样可以从数据流中更加有效的识别帧结构。

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;解析程序可以非常轻松的移植到其他规约上去。每一帧结构段的配置都可以单独进行测试和调试。在可维护性和可移植性上远胜于原来将解帧的工作集中到一个方法或一个类中的写法，在性能上也没有明显的下降。
解析方法ReadFrame3761并非和376规约耦合，可以不用修改的移植到其他规约的解析程序中，需要修改的是规约配置部分。待golang将来支持泛型之后，程序的可读性和扩展性还会得到很大改善。

&nbsp;&nbsp;&nbsp;&nbsp;类图
------------------------------
![此处输入图片的描述][1]
  
&nbsp;&nbsp;&nbsp;&nbsp;时序图
------------------------------
![此处输入图片的描述][2]


  [1]: https://github.com/jonenine/go3761/blob/master/doc/image/class.jpg
  [2]: https://github.com/jonenine/go3761/blob/master/doc/image/seq.jpg
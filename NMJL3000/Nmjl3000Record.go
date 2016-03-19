package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"github.com/tarm/serial"
	"golang.org/x/text/encoding/simplifiedchinese" //utf8 to gbk
	"golang.org/x/text/transform"
	"io/ioutil"
	"os"
	"strconv"
	"time"
)

//用到的常量
const (
	con_BAUD = 115200                 //要连接的串口波特率
	time_OUT = time.Millisecond * 500 //串口的超时时间
)

var con_CMDScan = []byte{0xA5, 0x5A, 0x00, 0x00, 0x0B, 0x00, 0x00, 0x00, 0xA0, 0xAA} //用于扫描端口时发送的命令含校验

var chanSerialPort = make(chan string, 1)
var goruntineDone = make(chan int, 1)
var gSerialPort string //全局变量用来放置扫描到的端口号

//----------------------------------------------------------------------------
//将4个byte类型合并为一个uint32类型,组合后b1，b2排列，如果是小端请自行调换位置
//------------------------------------------------------------------------------
func Fourbyte_to_uint32(b1 byte, b2 byte, b3 byte, b4 byte) uint32 {
	return uint32(b1)<<24 + uint32(b2)<<16 + uint32(b3)<<8 + uint32(b4) //bh左移8位再加上低位的bl
}

func Uint16_to_twobyte(i uint16) (byte, byte) {
	bh := byte(i >> 8)   //高位
	bl := byte(i & 0xff) //低位
	return bh, bl
}

//------------------------------------
//--uint32转换为4个byte
//----------------------------------
func Uint32_to_fourbyte(i uint32) (byte, byte, byte, byte) {
	bhh := byte(i >> 24) //高位
	bh := byte(i >> 16)  //低位
	bl := byte(i >> 8)
	bll := byte(i & 0xff) //低位
	return bhh, bh, bl, bll
}

//----------------------------------------------------------------------------
//将两个byte类型合并为一个uint16类型,组合后b1，b2排列，如果是小端请自行调换位置
//------------------------------------------------------------------------------
func Twobyte_to_uint16(bh byte, bl byte) uint16 {
	return uint16(bh)<<8 + uint16(bl) //bh左移8位再加上低位的bl
}

//-------------------------------------------------
//在计算byte的累加验证位，这是nm820采用的验证方式
//--------------------------------------------------
func sumCheck(date []byte) byte {
	var sum byte = 0x00
	for i := 0; i < len(date); i++ {
		sum = sum + date[i]
	}
	return sum
}

//-------------------------------------------------
//bcd码转10进制
//--------------------------------------------------
func decodeBcd(b byte) string {
	hi, lo := int(b>>4), int(b&0x0f)
	var x int
	x = int(10)*hi + lo
	return strconv.Itoa(x)
}

//-------------------------------------------------
//utf-8转为gbk编码
//--------------------------------------------------
func Utf8ToGbk(s []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GBK.NewEncoder())
	d, e := ioutil.ReadAll(reader)
	if e != nil {
		return nil, e
	}
	return d, nil
}

//================================================智能母猪饲喂器的历史记录结构体====================
/*
6. 每个控制器的记录
{
		每个控制器最多可存280条记录，每条记录16字节, 每个控制器记录占用空间 4608 字节, 每个控制器记录的
		首地址是

		(1<=n<=10)

		2816 + (n-1)*4608  				,控制器记录首地址
		2816 + (n-1)*4608 + 4500 	,记录控制器的地址，即 CZQPara.addr

		记录结构 (每条记录16字节)

		typedef struct
		{
		    U32 Ymdh;           //年月日时 0x15122911
		    U16 ms;             //分秒  0x4120
		    U16 type;           //类型 0 - 送料 ， 1 - 下料
		    U32 time;           //耗时s
		    U32 sum;            //校验和
		}__CZQRec;

}*/

type Nmjl3000History struct {
	Date     string //日期,和原始格式不同改为字符串
	Type     string //类型0--送料，1--下料,改动为字符串
	Time     uint32 //和原始格式不同改为分钟
	Sum      uint32 //校验，校验不正确，记录无效
	Addr     uint16 //地址，该记录隶属的控制器地址
	IsEffect string //根据前面的数据判断是否为有效记录
}

//用[]byte 更新他的值
func (h *Nmjl3000History) reflashValue(b []byte) {
	//年月日小时分钟秒
	h.Date = decodeBcd(b[3]) + "-" + decodeBcd(b[2]) + "-" + decodeBcd(b[1]) + "__" + decodeBcd(b[0]) + ":" + decodeBcd(b[5]) + ":" + decodeBcd(b[4])

	//记录类型
	ty := Twobyte_to_uint16(b[7], b[6])
	switch ty {
	case 0:
		h.Type = "送料"
	case 1:
		h.Type = "下料"
	default:
		h.Type = "未知"
	}

	//耗时
	h.Time = Fourbyte_to_uint32(b[11], b[10], b[9], b[8]) / 60

	//校验
	h.Sum = Fourbyte_to_uint32(b[15], b[14], b[13], b[12])
	var sumNum uint32               //传人字节数组的0-11的校验和
	for i := 0; i < len(b)-4; i++ { //去掉最后的校验u32
		sumNum = sumNum + uint32(b[i])
	}
	if (b[0] == 0) && (b[1] == 0) && (b[2] == 0) && (b[3] == 0) && (b[4] == 0) && (b[5] == 0) {
		h.IsEffect = "无效记录：时间为0"
		return
	}
	if sumNum != uint32(h.Sum) {
		h.IsEffect = "无效记录：和校验错误"
		return
	}
	h.IsEffect = "有效记录"
}

//结构体[]string化方便后面的csv输出
func (h *Nmjl3000History) toStrings() []string {
	var ss []string
	ss = append(ss, strconv.Itoa(int(h.Addr)))
	ss = append(ss, h.Date)
	ss = append(ss, h.Type)
	ss = append(ss, strconv.Itoa(int(h.Time)))
	ss = append(ss, h.IsEffect)
	return ss
}

//=======================================================================================

//向COMx发送命令，如果响应的前2个是我们的头协议0xa5,0x5a就是我们要用到的接口，并赋值到serialPort
func scanSerial(portnum int) {
	myserialPort := "COM" + fmt.Sprintf("%d", portnum)                             //拼接字符串
	c := &serial.Config{Name: myserialPort, Baud: con_BAUD, ReadTimeout: time_OUT} //超时500毫秒，如果500毫秒一过就会返回00，或者上一次的值
	s, err := serial.OpenPort(c)
	if err != nil {
		//fmt.Println("端口扫描：" + myserialPort + "不能打开或不存在")
		goruntineDone <- 1 //代表该线程完成了
		return
	}
	s.Write(con_CMDScan) //发送校验命令
	b1 := make([]byte, 1)
	b2 := make([]byte, 1)
	s.Read(b1) //如果没返回会超时返回00
	s.Read(b2)
	s.Close()
	//如果b1，b2 匹配包头0xa5,0x5a就可以断定是这个端口
	if (b1[0] == 0xa5) && (b2[0] == 0x5a) {
		fmt.Println("端口扫描：" + myserialPort + "为Nmlj3000料线控制器连接端口")
		chanSerialPort <- myserialPort
		goruntineDone <- 1 //代表该线程完成了
	} else {
		//fmt.Println("端口扫描：" + myserialPort + "不是连接端口")
		goruntineDone <- 1 //代表该线程完成了
	}
}

//===========================================
//获取3200条历史记录，需要用到全局变量
//出入参数为起始地址
//输出为历史记录结构体数组
//如果校验不成功将不放入History数组中
//startAddr 代表开始地址，long代表多多长的数据位
//U盘有最大读取限制不能大于2000的数据量，就是long不能大于2000
//=======================================
func read100History(startAddr uint32, long uint16) []*Nmjl3000History {
	//fmt.Println("===================发送命令得到历史记录========================")
	//获取历史记录的命令
	//               |包头     |   地址从4096开始      |      | 长度 1个结构体16
	//cmd1 := []byte{0xa5, 0x5a, 0x00, 0x00, 0x10, 0x00, 0x00, 0xc8, 0x00}
	if long > 1999 {
		fmt.Println("该U盘不能返回大于2000的数据包")
		return nil
	}
	if (long % 16) != 0 {
		fmt.Println("输入的长度必须为结构体的整数倍")
		return nil
	}

	cmdHead := []byte{0xa5, 0x5a}

	//为发送命令加上起始地址
	sabhh, sabh, sabl, sabll := Uint32_to_fourbyte(startAddr)
	cmd0 := append(cmdHead, []byte{sabhh, sabh, sabl, sabll}...)

	//为发送命令加上要读取的数据长度
	slbh, slbl := Uint16_to_twobyte(long)
	cmd1 := append(cmd0, []byte{0x00, slbh, slbl}...) //读取100条记录，字节为100*16

	//为发送命令加上校验和
	cmd2 := append(cmd1, sumCheck(cmd1))
	//fmt.Printf("%x\n", cmd2)
	//返回0xa5 0x5a addr3 addr2 addr1 addr0 0x80 lenH lenL Data Cs

	//发送串口命令
	c := &serial.Config{Name: gSerialPort, Baud: con_BAUD} //超时500毫秒，如果500毫秒一过就会返回00，或者上一次的值
	s, err := serial.OpenPort(c)
	if err != nil {
		fmt.Println("端口扫描：" + gSerialPort + "不能打开或不存在")
		return nil
	}
	s.Write(cmd2)

	b := make([]byte, 1)
	result := make([]byte, 10+int(long))
	//一共要读取的个数为10+数据长度，这里我们读10条数据就是10+160
	for i := 0; i < (10 + int(long)); i++ {
		s.Read(b)
		result[i] = b[0]
		//fmt.Printf("%x\n", b[0])
	}
	//fmt.Printf("%x\n", result[9:25])
	//fmt.Printf("%x\n", result)
	s.Close()

	//用结构体实例化，因为结果为16个字节。所以就是long/16个结构体
	var hs []*Nmjl3000History
	for i := 0; i < int(long)/16; i++ {
		h := &Nmjl3000History{}
		h.reflashValue(result[9+i*16 : 25+i*16])
		if h.IsEffect == "有效记录" {
			hs = append(hs, h) //如果是有效记录就加进去
		}
	}

	return hs
}

//读第n个控制器的地址(0-9)
func readCtrAddr(num int) uint16 {
	cmdHead := []byte{0xa5, 0x5a}
	//为发送命令加上起始地址
	////读第10部机地址
	//2816 + 9*4608 + 4500 = 0xBE94
	//A5 5A 00 00 BE 94 00 00 02 53
	//A5 5A 00 00 BE 94 80 00 02 19 00 EC //19 00 = 25
	startAddr := uint32(2816 + 4500 + num*4500)
	sabhh, sabh, sabl, sabll := Uint32_to_fourbyte(startAddr)
	cmd0 := append(cmdHead, []byte{sabhh, sabh, sabl, sabll}...)

	cmd1 := append(cmd0, []byte{0x00, 0x00, 0x02}...)

	//为发送命令加上校验和
	cmd2 := append(cmd1, sumCheck(cmd1))

	//发送串口命令
	c := &serial.Config{Name: gSerialPort, Baud: con_BAUD} //超时500毫秒，如果500毫秒一过就会返回00，或者上一次的值
	s, err := serial.OpenPort(c)
	if err != nil {
		fmt.Println("端口扫描：" + gSerialPort + "不能打开或不存在")
		return 9999
	}
	s.Write(cmd2)

	b := make([]byte, 1)
	result := make([]byte, 12)
	//一共要读取的个数为10+数据长度，这里我们读10条数据就是10+160
	for i := 0; i < 12; i++ {
		s.Read(b)
		result[i] = b[0]
		//fmt.Printf("%x\n", b[0])
	}
	s.Close()

	re := Twobyte_to_uint16(result[10], result[9])
	return re
}

//======================================
//输出为csv文件
//=======================================
func outCSV(hs []*Nmjl3000History) {
	currentTime := time.Now().Format("记录2006-01-02_15-04-02")
	name := currentTime + ".csv"
	file, _ := os.OpenFile(name, os.O_WRONLY|os.O_CREATE, os.ModePerm)
	w := csv.NewWriter(file)

	//将utf8转为gbk，表头
	headstring := []string{"控制器地址", "时间", "类型", "耗时单位分钟", "是否为有效记录"}
	for i := 0; i < len(headstring); i++ {
		gbk, err := Utf8ToGbk([]byte(headstring[i]))
		if err != nil {
			fmt.Println(err)
		}
		headstring[i] = string(gbk)
	}
	w.Write(headstring)

	//将结构体写到每一行
	for _, h := range hs { //将结构体数组打印过去
		bodystring := h.toStrings()

		for i := 0; i < len(bodystring); i++ {
			gbk, err := Utf8ToGbk([]byte(bodystring[i]))
			if err != nil {
				fmt.Println(err)
			}
			bodystring[i] = string(gbk)
		}
		w.Write(bodystring)
	}
	w.Flush()

	file.Close()
	fmt.Println("记录已输出到\"" + name + "\"文件夹下")

	/*下面的是读取
	rfile, _ := os.Open("test.csv")
	r := csv.NewReader(rfile)
	strs, _ := r.Read()
	for _, str := range strs {
		fmt.Print(str, "\t")
	}
	*/
}

func main() {
	fmt.Println("===========================================")
	fmt.Println("====       扫描端口COM1到COM20       ======")
	fmt.Println("===========================================")
	//用多线程进行端口扫描COM1到20
	for i := 1; i < 21; i++ {
		go scanSerial(i)
	}
	for i := 1; i < 21; i++ {
		<-goruntineDone //确保所有的线程都运行完毕
	}
	select {
	case gSerialPort = <-chanSerialPort: //赋值到全局变量
		fmt.Println("扫描成功：" + gSerialPort)
		fmt.Println("")
		fmt.Println("===========================================")
		fmt.Println("====    发送命令获取1-3200号历史记录    ===")
		fmt.Println("===========================================")

		//每组读取100条数据记录，一个读取32次，就是一个读取3200条记录
		var hs []*Nmjl3000History
		fmt.Println("开始读取，读取的数据量比较大，需要20秒左右....")
		fmt.Println("完成时间：|--------------------------------")
		fmt.Print("当前进度：|")
		//读10个控制器，每个控制器分3次读，每个控制器占用4608字节
		for i := 0; i < 10; i++ {
			//读出第一个控制器的地址
			addr := readCtrAddr(i)
			htemp := read100History(2816+uint32(i)*4608, 16*100) //先读一百条
			//为读出的结构体都赋值为这个地址
			for _, eveyH := range htemp {
				eveyH.Addr = addr
			}
			hs = append(hs, htemp...)
			fmt.Print("=")
			htemp = read100History(2816+uint32(i)*4608+16*100, 16*100) //再读一百条
			for _, eveyH := range htemp {
				eveyH.Addr = addr
			}
			hs = append(hs, htemp...)
			fmt.Print("=")
			htemp = read100History(2816+uint32(i)*4608+16*200, 16*80) //最后读80条
			for _, eveyH := range htemp {
				eveyH.Addr = addr
			}
			hs = append(hs, htemp...)
			fmt.Print("=")
		}
		fmt.Println("")
		fmt.Println("")
		//将hs数组输出为CSV
		fmt.Println("===========================================")
		fmt.Println("====         将记录输出到CSV文件        ===")
		fmt.Println("===========================================")
		outCSV(hs)
		fmt.Println("按回车键退出..........")
		reader := bufio.NewReader(os.Stdin)
		reader.ReadString('\n')
		return //输出结构体数组到CSV中
		//fmt.Println(hs)

	case <-time.After(3 * time.Second):
		fmt.Println("错误：不能扫描到端口，请检测U盘是否插上，并安装上驱动！")
		fmt.Println("按回车键退出..........")
		reader := bufio.NewReader(os.Stdin)
		reader.ReadString('\n')
		return
		//fmt.Printf("Input Char Is : %v", string([]byte(input)[0]))
	}

}

package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"github.com/tarm/serial"
	"golang.org/x/text/encoding/simplifiedchinese" //utf8 to gbk
	"golang.org/x/text/transform"                  //utf8 to gbk
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

var chanSerialPort = make(chan string, 1)
var goruntineDone = make(chan int, 1)
var gSerialPort string //全局变量用来放置扫描到的端口号

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

//================================================智能母猪饲喂器的历史记录结构体====================

type History struct {
	PigNum   uint32 //猪耳标号
	Date     string //时标，从1970-01-01 00:00:00到现在的秒数，时标为0或者0xffffffff为无效,原始数值为uint32
	Amount   uint16 //可以吃的总量
	Actual   uint16 //实际吃的数量
	PN1      byte
	PN2      byte
	Sum      byte   //校验，校验不正确，记录无效
	IsEffect string //根据前面的数据判断是否为有效记录
}

//用[]byte 更新他的值
func (h *History) reflashValue(b []byte) {
	h.PigNum = Fourbyte_to_uint32(b[3], b[2], b[1], b[0])
	//将Unix时间转为字符串
	t := Fourbyte_to_uint32(b[7], b[6], b[5], b[4])
	//如果时间为ff ff ff ff为无效记录

	//fmt.Println(t)
	t = t - 8*60*60                                               //t为格林尼治时间，减去8小时就为北京时间
	h.Date = time.Unix(int64(t), 0).Format("2006-01-02 15:04:05") //时间

	h.Amount = Twobyte_to_uint16(b[9], b[8])
	h.Actual = Twobyte_to_uint16(b[11], b[10])
	h.PN1 = b[12]
	h.PN2 = b[13]
	h.Sum = b[15]
	//时间为ffffffff 和0都是无效的记录
	if b[7] == 0xff && b[6] == 0xff && b[5] == 0xff && b[4] == 0xff {
		h.IsEffect = "无效记录：时间段错误"
		return
	}
	if b[7] == 0x00 && b[6] == 0x00 && b[5] == 0x00 && b[4] == 0x00 {
		h.IsEffect = "无效记录：时间段错误"
		return
	}
	//因为校验和为uint32所以全部转为uint32,将每一个字节转换为转换为uint32
	var sumNum byte //传人字节数组的0-11的校验和
	for i := 0; i < 14; i++ {
		sumNum = sumNum + b[i]
	}
	if sumNum != h.Sum {
		h.IsEffect = "无效记录：和校验错误"
		return
	}
	h.IsEffect = "有效记录"
}

//结构体[]string化方便后面的csv输出
func (h *History) toStrings() []string {
	var ss []string
	ss = append(ss, strconv.Itoa(int(h.PigNum)))
	ss = append(ss, h.Date)
	ss = append(ss, strconv.Itoa(int(h.Amount)))
	ss = append(ss, strconv.Itoa(int(h.Actual)))
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
	cmd := []byte{0xa5, 0x5a, 0x00, 0x00, 0x00, 0x80, 0x00, 0x00, 0x10, 0x8f} //随便的一条命令，如果是正确的端口有返回
	s.Write(cmd)
	b1 := make([]byte, 1)
	b2 := make([]byte, 1)
	s.Read(b1) //如果没返回会超时返回00
	s.Read(b2)
	s.Close()
	//如果b1，b2 匹配包头0xa5,0x5a就可以断定是这个端口
	if (b1[0] == 0xa5) && (b2[0] == 0x5a) {
		fmt.Println("端口扫描：" + myserialPort + "为智能母猪饲喂器连接端口")
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
//=======================================
func read100History(startAddr uint32, long uint16) []*History {
	//fmt.Println("===================发送命令得到历史记录========================")
	//获取历史记录的命令
	//               |包头     |   地址从4096开始      |      | 长度 1个结构体16
	//cmd1 := []byte{0xa5, 0x5a, 0x00, 0x00, 0x10, 0x00, 0x00, 0xc8, 0x00}
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
	result := make([]byte, 10+100*16)
	//一共要读取的个数为10+数据长度，这里我们读10条数据就是10+160
	for i := 0; i < (10 + 100*16); i++ {
		s.Read(b)
		result[i] = b[0]
		//fmt.Printf("%x\n", b[0])
	}
	//fmt.Printf("%x\n", result[9:25])
	//fmt.Printf("%x\n", result)
	s.Close()

	//用结构体实例化，连续10次
	var hs []*History
	for i := 0; i < 100; i++ {
		h := &History{}
		h.reflashValue(result[9+i*16 : 25+i*16])
		if h.IsEffect == "有效记录" {
			hs = append(hs, h) //如果是有效记录就加进去
		}
	}

	return hs
}

//======================================
//输出为csv文件
//=======================================
func outCSV(hs []*History) {
	currentTime := time.Now().Format("记录2006-01-02_15-04-02")
	name := currentTime + ".csv"
	file, _ := os.OpenFile(name, os.O_WRONLY|os.O_CREATE, os.ModePerm)
	w := csv.NewWriter(file)

	//将utf8转为gbk，表头
	headstring := []string{"猪耳标号", "日期", "额定吃料量", "实际吃料量", "是否为有效记录"}
	for i := 0; i < len(headstring); i++ {
		gbk, err := Utf8ToGbk([]byte(headstring[i]))
		if err != nil {
			fmt.Println(err)
		}
		headstring[i] = string(gbk)
	}
	w.Write(headstring)

	//将结构体输出
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
		var hs []*History
		fmt.Println("开始读取，读取的数据量比较大，需要20秒左右....")
		fmt.Println("完成时间：|--------------------------------")
		fmt.Print("当前进度：|")
		for i := 0; i < 32; i++ {
			htemp := read100History(4096+uint32(i)*1600, 1600) //结构体数组,读取1-100号记录，返回的数组是有效记录数组
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

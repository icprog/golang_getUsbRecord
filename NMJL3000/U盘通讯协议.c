波特率 115200 , 8N1

地址范围 0 - 256KB (0 - 0x3FFFF)
每次读写长度最多 2048 Byte

1. 读
{		
							0    1    2     3     4     5     6    7    8    9
	 PC发送: 		0xA5 0x5A Addr3 Addr2 Addr1 Addr0 0x00 LenH LenL CS
	 回复:   		0xA5 0x5A Addr3 Addr2 Addr1 Addr0 0x80 LenH LenL Data CS
	 回复错误:  0xA5 0x5A Addr3 Addr2 Addr1 Addr0 0xC0 ErrCode CS	
} 

2. 写
{
							0    1		2			3			4			5			6		 7    8    9    9+dlen
	 PC发送: 		0xA5 0x5A Addr3 Addr2 Addr1 Addr0 0x01 LenH LenL Data CS
	 回复: 		  0xA5 0x5A Addr3 Addr2 Addr1 Addr0 0x81 LenH LenL CS	
	 回复错误:  0xA5 0x5A Addr3 Addr2 Addr1 Addr0 0xC1 ErrCode CS	
}

3. 错误代码
{
	  1  -  地址超范围
	  2  -  长度超范围
}


4. 例
{
		//写
		S: A5 5A 00 00 00 00 01 00 10 01 02 03 04 05 06 07 08 09 10 11 12 13 14 15 16 C2
		R: A5 5A 00 00 00 00 81 00 10 90 
			
		S: A5 5A 00 01 00 00 01 00 10 11 22 33 44 55 66 77 88 99 00 21 22 23 24 25 26 E3
		R: A5 5A 00 01 00 00 81 00 10 91 
		
		S: A5 5A 00 02 00 00 01 00 10 21 22 23 24 25 26 27 28 29 20 21 22 23 24 25 26 54
		R: A5 5A 00 02 00 00 81 00 10 92 
			
		S: A5 5A 00 03 00 00 01 00 10 31 32 33 34 35 36 37 38 39 30 31 32 33 34 35 36 55
		R: A5 5A 00 03 00 00 81 00 10 93 
		
		S: A5 5A 00 00 FF F5 01 00 10 AA BB CC DD EE FF 11 22 33 44 55 66 77 88 99 00 FC
		R: A5 5A 00 00 FF F5 81 00 10 84 
		
		//读
		S: A5 5A 00 00 00 00 00 00 10 0F
		R: A5 5A 00 00 00 00 80 00 10 01 02 03 04 05 06 07 08 09 10 11 12 13 14 15 16 41 
			
		S: A5 5A 00 01 00 00 00 00 10 10
		R: A5 5A 00 01 00 00 80 00 10 11 22 33 44 55 66 77 88 99 00 21 22 23 24 25 26 62 
			
		S: A5 5A 00 02 00 00 00 00 10 11
		R: A5 5A 00 02 00 00 80 00 10 21 22 23 24 25 26 27 28 29 20 21 22 23 24 25 26 D3 
			
		S: A5 5A 00 03 00 00 00 00 10 12
		R: A5 5A 00 03 00 00 80 00 10 31 32 33 34 35 36 37 38 39 30 31 32 33 34 35 36 D4
			
		S: A5 5A 00 00 FF F5 00 00 10 03
		R: A5 5A 00 00 FF F5 80 00 10 AA BB CC DD EE FF 11 22 33 44 55 66 77 88 99 00 7B 
}

5. 每个控制器记录内容
{
		每个控制器占256字节长度的内容，即每个控制器开始的地址为256*n (n<=10)
		typedef struct
		{
		    U16 slMode;                 //送料模式 0 = 停止, 1 = 启动, 2 = 自动
		    U16 xlMode;                 //下料模式 0 = 停止, 1 = 启动, 2 = 自动
		    U16 test;                   //测试模式 0 = 关闭, 1 = 启用
		    U16 tCurrent;               //测试电流
		    U16 slTab[MAX_TIMETAB];     //自动送料时间表
		    U16 xlTab[MAX_TIMETAB];     //自动下料时间表
		    U16 tabNum;                 //<=20,默认2
		    U16 jlStartDelay;           //绞龙启动延迟 mmss
		    U16 detectDelay;            //料满开关延迟 mmss
		    U16 clear;                  //自动清洁定量杯 0 = 否 , 1 = 是
		    U16 usePass;                //是否使用密码 0 = 否 , 1 = 是
		    U16 xlOpenTime;             //下料电机打开时间 mmss
		    U16 xlKeepTime;             //下料电机保持时间 mmss
		    U16 xlCloseTime;            //下料电机关闭时间 mmss
		    U16 xlSensor;               //下料限位 0 = 不使用 1 = 使用
		    U16 slStopDelay;            //塞链停止延迟时间 mmss
		    U16 slMaxTime;              //塞链最大运行时间 hhmm
		    U16 slMaxCTime;             //塞链最大电流运行时间 mmss
		    U16 slMaxCurrent;           //塞链最大电流 20.0A
		    U16 slMaxCD;                //塞链最大电流回差 9.9A
		    U16 slAMaxCurrent;          //塞链极限电流 20.0A
		    U16 slAMaxCTime;            //塞链极限电流时间 mmss
		    U16 idleCurrent;            //空载电流
		    U16 idleDelay;              //空载延迟
		    U16 addr;                   //控制器地址
		    U32 password;               //控制器密码
		    U32 lastTime;               //上1次耗时
		    U32 todayTime;              //今天总耗时
		    U32 timeHis[5];             //上5天总耗时
		    U32 Sum;
		}__CZQPara;	
}
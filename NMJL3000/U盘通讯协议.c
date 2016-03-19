������ 115200 , 8N1

��ַ��Χ 0 - 256KB (0 - 0x3FFFF)
ÿ�ζ�д������� 2048 Byte

1. ��
{		
							0    1    2     3     4     5     6    7    8    9
	 PC����: 		0xA5 0x5A Addr3 Addr2 Addr1 Addr0 0x00 LenH LenL CS
	 �ظ�:   		0xA5 0x5A Addr3 Addr2 Addr1 Addr0 0x80 LenH LenL Data CS
	 �ظ�����:  0xA5 0x5A Addr3 Addr2 Addr1 Addr0 0xC0 ErrCode CS	
} 

2. д
{
							0    1		2			3			4			5			6		 7    8    9    9+dlen
	 PC����: 		0xA5 0x5A Addr3 Addr2 Addr1 Addr0 0x01 LenH LenL Data CS
	 �ظ�: 		  0xA5 0x5A Addr3 Addr2 Addr1 Addr0 0x81 LenH LenL CS	
	 �ظ�����:  0xA5 0x5A Addr3 Addr2 Addr1 Addr0 0xC1 ErrCode CS	
}

3. �������
{
	  1  -  ��ַ����Χ
	  2  -  ���ȳ���Χ
}


4. ��
{
		//д
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
		
		//��
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

5. ÿ����������¼����
{
		ÿ��������ռ256�ֽڳ��ȵ����ݣ���ÿ����������ʼ�ĵ�ַΪ256*n (n<=10)
		typedef struct
		{
		    U16 slMode;                 //����ģʽ 0 = ֹͣ, 1 = ����, 2 = �Զ�
		    U16 xlMode;                 //����ģʽ 0 = ֹͣ, 1 = ����, 2 = �Զ�
		    U16 test;                   //����ģʽ 0 = �ر�, 1 = ����
		    U16 tCurrent;               //���Ե���
		    U16 slTab[MAX_TIMETAB];     //�Զ�����ʱ���
		    U16 xlTab[MAX_TIMETAB];     //�Զ�����ʱ���
		    U16 tabNum;                 //<=20,Ĭ��2
		    U16 jlStartDelay;           //���������ӳ� mmss
		    U16 detectDelay;            //���������ӳ� mmss
		    U16 clear;                  //�Զ���ඨ���� 0 = �� , 1 = ��
		    U16 usePass;                //�Ƿ�ʹ������ 0 = �� , 1 = ��
		    U16 xlOpenTime;             //���ϵ����ʱ�� mmss
		    U16 xlKeepTime;             //���ϵ������ʱ�� mmss
		    U16 xlCloseTime;            //���ϵ���ر�ʱ�� mmss
		    U16 xlSensor;               //������λ 0 = ��ʹ�� 1 = ʹ��
		    U16 slStopDelay;            //����ֹͣ�ӳ�ʱ�� mmss
		    U16 slMaxTime;              //�����������ʱ�� hhmm
		    U16 slMaxCTime;             //��������������ʱ�� mmss
		    U16 slMaxCurrent;           //���������� 20.0A
		    U16 slMaxCD;                //�����������ز� 9.9A
		    U16 slAMaxCurrent;          //�������޵��� 20.0A
		    U16 slAMaxCTime;            //�������޵���ʱ�� mmss
		    U16 idleCurrent;            //���ص���
		    U16 idleDelay;              //�����ӳ�
		    U16 addr;                   //��������ַ
		    U32 password;               //����������
		    U32 lastTime;               //��1�κ�ʱ
		    U32 todayTime;              //�����ܺ�ʱ
		    U32 timeHis[5];             //��5���ܺ�ʱ
		    U32 Sum;
		}__CZQPara;	
}
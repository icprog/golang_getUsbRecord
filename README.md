中山BW800饲喂器 NMJL3000料线控制器 U盘历史记录读取
====================================================


用到的关键代码
-------------

1. utf转gbk（execel里面需要gbk编码的string）

```go
	for i := 0; i < len(bodystring); i++ {
		gbk, err := Utf8ToGbk([]byte(bodystring[i]))
		if err != nil {
			fmt.Println(err)
		}
		bodystring[i] = string(gbk)
	}
```

2. 将utc时间字符串化，北京时间

```go
	t := Fourbyte_to_uint32(b[7], b[6], b[5], b[4])
	t = t - 8*60*60                                               //t为格林尼治时间，减去8小时就为北京时间
	h.Date = time.Unix(int64(t), 0).Format("2006-01-02 15:04:05") //时间
```
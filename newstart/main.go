package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"

	"G:\区块链\开源挖矿软件\newstart\mod\cl"
	"G:\区块链\开源挖矿软件\newstart\mod\mining"
	"G:\区块链\开源挖矿软件\newstart\mod\sia"
)

var Version = "天悔1.0.0"

// 开发的版本为1.0.0，天悔第一版

var intensity = 28

// 挖矿的强度为28，intensity中文翻译为强度

var devicesTypeSForMining = cl.DeviceTypeGPU

//挖矿的设备类型，devicesTypeSForMining中文翻译为设备用于挖掘的类型
//cl库为GO的OpenCL绑定，查看地址为https://github.com/robvanmieghem/go-opencl/tree/master/cl
//cl.DeviceTypeGPU这个就是代表指定使用GPU进行挖矿
//GPU就是显卡的核心部分

func main() {
	log.SetOutput(os.Stdout)
	//将日志输出标准化，可以使用log.Println("This is a log message.")标准化打印日志
	//2023/11/15 01:44:45 This is a log message.

	printVersion := flag.Bool("v", false, "显示版本并退出")
	//flag包是处理命令行参数的go中自带的包，这个意思就是命令行输入-v会提示显示版本并退出，默认是false，就是如果不输入那么就不会显示

	useCPU := flag.Bool("cpu", false, "如果设置，也使用CPU进行挖掘，默认情况下仅使用GPU")
	//同上-cpu

	host := flag.String("url", "localhost:9980", "守护进程或服务器主机和端口，对于stratum服务器，使用“stratum+tcp://<host>：<port>”`")
	//如果用户不输入-url，那么默认是localhost:9980，flag.String和flag.Bool的区别在于默认值一个是bool类型一个是string类型

	pooluser := flag.String("user", "payoutaddress.rigname", "大多数stratum服务器都采用[钱包地址].[矿工名]")
	//一般输入的都是payoutaddress.rigname

	excludeGPUs := flag.String("E", "", "设置要排除的GPU列表，以逗号分隔的设备编号列表")
	//设置要排除的GPU设备列表':'分割

	flag.IntVar(&intensity, "I", intensity, "难度设置")
	//flag.IntVar的含义就是返回值为int
	//也可以这么写intensity :=flag.Int("I",28,"设置挖矿难度，默认为28")，但是要删除上面的var intensity = 28

	flag.Parse()
	//作用为解析命令行参数

	if *printVersion {
		fmt.Println("版本为", Version)
		os.Exit(0)
	}
	//如果用户输入了-v，那么就会返回版本信息
	//os.Exit(0)则是正常退出程序，如果os.Exit(1)就是异常退出程序

	if *useCPU {
		devicesTypeSForMining = cl.DeviceTypeAll

	}
	//如果用户输入了-cpu，那么就会使用所有可用设备进行挖矿，不输入的话默认是GPU

	globalItemSize := int(math.Exp2(float64(intensity)))
	//从里往外讲解，float64是将intensity参数转换为浮点数类型，math.Exp2是计算2的指数，就是2的float64(intensity)次方
	//int是将math.Exp2计算后的结果转换为整数
	//计算挖矿中的全局项目大小，用于配置挖矿算法的参数

	platforms, err := cl.GetPlatforms()
	if err != nil {
		log.Panic(err)
		//以日志形式输出错误
	}
	//获取系统上可用的 OpenCL 平台列表，获取支持的 OpenCL 平台是为了确定系统上可以用于挖矿的GPU

	clDevices := make([]*cl.Device, 0, 4)
	//制作一个切片存储OpenCL设备列表

	for _, platform := range platforms {
		log.Println("OpenCL平台设备列表名为：", platform.Name())
		//记录每个设备到日志当中
		platormDevices, err := cl.GetDevices(platform, devicesTypeSForMining)
		if err != nil {
			log.Panicln(err)
		}
		//使用 cl.GetDevices 函数获取该平台上支持的设备列表

		log.Panicln(len(platormDevices), "找到设备为：")
		for i, device := range platormDevices {
			log.Panicln(i, "-", device.Type(), "-", device.Name())
			clDevices = append(clDevices, device)
		}
		//对于每个设备，程序记录设备的类型、名称等信息到日志中

	}

	if len(clDevices) == 0 {
		log.Panicln("没有找到可用的设备")
		os.Exit(1)
	}
	//如果切片中没有找到可用的设备，那么就会异常退出

	miningDevices := make(map[int]*cl.Device)
	//创建一个空的 map，其中键是整数，值是指向 cl.Device 类型的指针
	for i, device := range miningDevices {
		//使用 range 遍历 miningDevices 中的每个键值对，其中 i 是键，device 是对应的值
		if deviceExcludedForMining(i, *excludeGPUs) {
			//利用deviceExcludedForMining函数，如果为true就退出本次循环，不加入这个运行列表
			continue
		}
		miningDevices[i] = device
		//如果为false，就会加入miningDevices这个map中
	}

	nrOFMiningDevices := len(miningDevices)
	//计算可以挖矿的GPU数量
	var hashRateReportsChannel = make(chan *mining.HashRateReport, nrOFMiningDevices*10)
	// 创建了一个通道（channel），用于传递挖矿的哈希率报告。通道的类型是 *mining.HashRateReport，并且设置了缓冲区的大小为 nrOfMiningDevices*10

	var miner mining.Miner
	log.Panicln("启动SIA挖掘")
	c := sia.NewClient(*host, *pooluser)
	//创建一个 sia 包中的客户端实例，并将其地址赋值给变量 c。函数的参数包括 host 和 pooluser，用于指定挖矿池的地址和用户信息
	miner = &sia.Miner{
		//定义了一个sia下Miner的结构体，在algorithms\sia\miner.go中
		CLDevices:       miningDevices,
		HashRateReports: hashRateReportsChannel,
		Intensity:       intensity,
		GlobalItemSize:  globalItemSize,
		Client:          c,
	}
	miner.Miner()

	hashRateReports := make([]float64, nrOFMiningDevices)
	//创建一个长度为 nrOFMiningDevices 的 float64 类型切片，用于存储每个矿工的哈希率报告
	for {
		for i := 0; i < nrOFMiningDevices; i++ {
			report := <-hashRateReportsChannel
			hashRateReports[report.MinerID] = report.HashRate
		}
		fmt.Print("\r")
		//进入无限循环，该循环从 hashRateReportsChannel 通道中接收矿工的哈希率报告，并更新 hashRateReports 中对应矿工的哈希率。
		var totalHashRate float64
		for minerID, hashrate := range hashRateReports {
			fmt.Printf("%d-%.1f", minerID, hashrate)
			totalHashRate += hashrate
		}
		//在内部循环结束后，计算总的哈希率。遍历 hashRateReports 中的每个矿工，打印矿工ID和对应的哈希率
		fmt.Printf("运算总数为：%.1f MH/s", totalHashRate)
		//打印总的哈希率，以 MH/s（兆哈希每秒）为单位。这里使用 Printf 函数格式化输出。
	}

}

func deviceExcludedForMining(deviceID int, excludedGPUs string) bool {
	//deviceID 表示设备ID，另一个是字符串类型的 excludedGPUs 表示要排除的GPU列表。
	excludedGPUList := strings.Split(excludedGPUs, ",")
	//strings.Split作用是将切片中每一个按照，号分割成一个切片
	for _, excludedGPU := range excludedGPUList {
		//利用for循环取出列表中需要排除的GPU
		if strconv.Itoa(deviceID) == excludedGPU {
			//strconv.Itoa将deviceID转换为字符串类型，如果deviceID等于要排除的GPU那么就返回true
			return true
		}
	}
	return false
}

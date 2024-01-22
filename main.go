package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/sirupsen/logrus"
)

// 全局的 http.Client 实例，包含连接池配置
var (
	client = &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        240,
			MaxIdleConnsPerHost: 60,
		},
	}
	MintCount    atomic.Uint64
	Prefix       string
	Challenge    string
	HexAddresses []string
	wg           sync.WaitGroup
	mutex        sync.Mutex
)

// 用于存储交易请求信息的结构体
type TransactionRequest struct {
	Body string
}

// 初始化函数，获取输入的地址和难度
func init() {
	Challenge = "72424e4200000000000000000000000000000000000000000000000000000000"
	log.SetFormatter(&log.TextFormatter{TimestampFormat: "15:04:05", FullTimestamp: true})

	fmt.Print("请输入多个地址，按回车结束输入:\n")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	Addresses := scanner.Text()

	addressList := strings.Fields(Addresses)
	for _, addr := range addressList {
		addr = strings.ToLower(strings.TrimPrefix(addr, "0x"))
		HexAddresses = append(HexAddresses, "0x"+addr)
	}

	fmt.Print("请输入难度：")
	fmt.Scanln(&Prefix)
}

func main() {
	// 定时器，每隔 30 分钟执行一次
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	// 交易请求通道
	txChannel := make(chan *TransactionRequest, 60)
	defer close(txChannel)

	// 遍历地址并启动多个 generateTx 和 validateTx 协程
	for _, hexAddr := range HexAddresses {
		for i := 0; i < 10; i++ {
			wg.Add(2)
			go generateTx(hexAddr, txChannel)
			go validateTx(hexAddr, txChannel)
		}
	}

	// 信号处理，捕捉 Ctrl+C 信号
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		os.Exit(0)
	}()

	// 等待所有协程完成
	wg.Wait()
}

// 生成交易协程
func generateTx(hexAddr string, txChannel chan<- *TransactionRequest) {
	defer wg.Done()

	for {
		randomValue := make([]byte, 32)
		_, err := rand.Read(randomValue)
		if err != nil {
			log.Error(err)
			return
		}

		potentialSolution := hex.EncodeToString(randomValue)
		address64 := fmt.Sprintf("%064s", strings.ToLower(strings.TrimPrefix(hexAddr, "0x")))
		dataTemps := fmt.Sprintf(`%s%s%s`, potentialSolution, Challenge, address64)

		dataBytes, err := hex.DecodeString(dataTemps)
		if err != nil {
			log.Error(err)
			return
		}

		hashedSolutionBytes := crypto.Keccak256(dataBytes)
		hashedSolution := fmt.Sprintf("0x%s", hex.EncodeToString(hashedSolutionBytes))

		if strings.HasPrefix(hashedSolution, Prefix) {
			// 使用互斥锁保护共享资源
			mutex.Lock()
			MintCount.Add(1)
			mutex.Unlock()

			// 如果日志级别是 Info，打印找到新ID的信息
			if log.GetLevel() == log.InfoLevel {
				log.WithFields(log.Fields{"Solution": hashedSolution}).Info("找到新ID")
			}

			// 构造交易请求信息并发送到通道
			body := fmt.Sprintf(`{"solution": "0x%s", "challenge": "0x%s", "address": "%s", "difficulty": "%s", "tick": "%s"}`, potentialSolution, Challenge, strings.ToLower(hexAddr), Prefix, "rBNB")
			txChannel <- &TransactionRequest{Body: body}
		}
	}
}

// 验证交易协程
func validateTx(hexAddr string, txChannel chan *TransactionRequest) {
	defer wg.Done()

	for {
		// 从通道中获取交易请求
		txRequest := <-txChannel

		// 构造 HTTP 请求
		req, err := http.NewRequest("POST", "https://ec2-18-218-197-117.us-east-2.compute.amazonaws.com/validate", strings.NewReader(txRequest.Body))
		if err != nil {
			log.Error(err)
			return
		}

		// 设置请求头
		req.Header.Set("content-type", "application/json")
		req.Header.Set("origin", "https://bnb.reth.cc")
		req.Header.Set("referer", "https://bnb.reth.cc/")
		req.Header.Set("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

		// 发送 HTTP 请求
		resp, err := client.Do(req)
		if err != nil {
			log.Error(err)
			return
		}

		// 关闭响应体
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				log.Error(err)
				return
			}
		}(resp.Body)

		// 读取响应体内容
		bodyText, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Error(err)
			return
		}

		// 将响应体内容转换为字符串
		bodyString := string(bodyText)

		// 检查响应中是否包含 "validate success!"，表示验证成功
		containsValidateSuccess := strings.Contains(bodyString, "validate success!")
		if containsValidateSuccess {
			// 如果验证成功，打印 MINT 成功的信息
			log.WithFields(log.Fields{"Address": strings.ToLower(hexAddr)}).Info("MINT成功")
		} else {
			// 如果验证失败，打印 MINT 错误的信息
			log.WithFields(log.Fields{"错误": err}).Error("MINT错误")
		}
	}
}

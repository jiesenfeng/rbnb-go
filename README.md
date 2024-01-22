批量rbnb-go
-----------------------
### 下载之后安装如果电脑自带环境可以跳过这一步

### **Go下载地址:**https://golang.org/dl/ 

### Git下载地址:https://git-scm.com/downloads 

### **在解压后的rBNB脚本文件夹内运行**CMD 分别执行以下命令：

```shell
git clone 
go build 
```
## **输入多个钱包的时候每个钱包地址用空格隔开**



添加了全局的 `client` 变量，用于整个应用程序生命周期内的 HTTP 请求，这个全局变量将在整个程序中被重复使用，确保了连接池的有效管理和重用。并在适当的地方使用了互斥锁 `mutex` 来保护共享资源。

并发地进行地址生成和验证交易，每个地址都会启动多个 `generateTx` 和 `validateTx` 协程。这种并发的设计允许程序同时处理多个地址的交易生成和验证，提高了程序的性能。

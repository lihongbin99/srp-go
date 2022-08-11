# Srp-Go

srp(simple reverse proxy) 是一个简单的内网穿透反向代理应用, 支持 TCP、UDP. 可以将内网服务通过具有公网 IP 的服务器暴露到公网。

## 使用方法
1. 启动服务端 / 客户端(会尝试读取可执行文件同目录下的config.ini)
    - 服务端
        ```shell
        ./srp
        ./srp | tee srp.log
        nohup ./srp >> srp.log 2>&1 &
        ```
    - 客户端
        ```shell
        start /min .\srp.exe
        ```
2. 指定配置文件启动服务端 / 客户端
    - 服务端
        ```shell
        ./srp | tee log.txt -c config\server.ini
        ./srp -c config\server.ini | tee srp.log
        nohup ./srp -c config\server.ini >> srp.log 2>&1 &
        ```
    - 客户端
        ```shell
        start /min .\srp.exe -c config\client.ini
        ```

### 配置文件

服务端配置文件 `server.ini`：
```ini
[Common]
; 服务器的ip, 不设置可以同时监听ipv4和ipv6
Listen.Ip   =
; 服务器的端口
Listen.Port = 13520

; 配置日志级别: [info, debug, trace]
Log.Level = info

; 是否开启安全传输
Security.Enable     = true
; SSL私钥文件地址(当security.enable=true时必须设置)
Security.PrivateKey = config\security\private.pem

; 超时设置(秒)
KeepAlive.Interval = 30
KeepAlive.Timeout  = 10
```

客户端配置文件 `client.ini`：
```ini
[Common]
; 本机名称(每个客户端需要设置不同的名称)(必须)
Client.Name = client
; 服务器的ip
Server.Ip   = 0.0.0.0
; 服务器的端口
Server.Port = 13520

; 配置日志级别: [info, debug, trace]
Log.Level = trace

; 是否开启安全传输
Security.Enable    = true
; SSL公钥地址(当security.enable=true时必须设置)
Security.PublicKey = config\security\public.pem

; 超时设置(秒)
KeepAlive.Interval = 30
KeepAlive.Timeout  = 10

; 以 P2p- 为前缀即可
[P2p-1]
; 传输协议 [tcp udp tcp,udp]
Protocol   = tcp,udp
; 本地监听端口
LocalPort  = 23521
; 远程名称
TargetName = cs
; 远程端口
TargetPort = 28080
[P2p-2]
Protocol   = tcp,udp
LocalPort  = 23522
TargetName = cs
TargetPort = 28080

; 以 Proxy- 为前缀即可
[Proxy-1]
; 传输协议 [tcp udp tcp,udp]
Protocol   = tcp,udp
; 本地监听端口
LocalPort  = 23523
; 远程名称
TargetName = cs
; 远程IP(如果不设置的话默认连接ipv4或ipv6)
TargetIp   =
; 远程端口
TargetPort = 28080
[Proxy-2]
Protocol   = tcp,udp
LocalPort  = 23524
TargetName = cs
TargetIp   =
TargetPort = 28080

; 以 Nat- 为前缀即可
[Nat-1]
; 传输协议 [tcp udp tcp,udp]
Protocol   = tcp,udp
; 本地端口
LocalPort  = 23525
; 远程名称
TargetName = cs
; 远程IP(如果不设置的话默认连接ipv4或ipv6)
TargetIp   =
; 服务器监听端口
RemotePort = 23526
[Nat-2]
Protocol   = tcp,udp
LocalPort  = 23527
TargetName = cs
TargetIp   =
RemotePort = 23528
```

客户端`p2p`配置：
```text
使用点对点传输
  -- 需要双方都启动客户端
  -- 不需要在服务器开启端口
  -- 流量不经过服务器
  -- 配置在发送端
  -- 不一定成穿透成功
```

客户端`proxy`配置：
```text
使用服务器代理传输流量
  -- 需要双方都启动客户端
  -- 不需要在服务器开启端口
  -- 流量经过服务器
  -- 配置在发送端
```

客户端`nat`配置：
```text
使用服务器开启端口监听代理传输流量
  -- 只需要一方启动客户端
  -- 需要在服务器开启端口
  -- 流量经过服务器
  -- 配置在接收端
```

###服务端控制台
```text
e 退出程序
r 更新配置
c 查看连接详情
```
###客户端控制台
```text
e 退出程序
n 通知其他客户端, 依次输入 客户端名称 操作类型(e, r) 参数(可选)
o 通知服务器, 依次输入 操作类型(e, r, c) 参数(可选)
r 更新配置
c 查看连接详情
```

## 构建
开发环境
```shell
go build -tags idea
```
编译适用于 64 位 Linux 操作系统的服务端可执行文件：
```shell
SET CGO_ENABLED=0
SET GOOS=linux
go build -o ./bin/srp ./main/server
```
编译适用于 64 位 Windows 操作系统的客户端可执行文件：
```shell
SET CGO_ENABLED=1
SET GOOS=windows
go build -o ./bin/srp.exe ./main/client
```

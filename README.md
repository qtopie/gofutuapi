# go-futu-api

[富途牛牛OpenD](https://openapi.futunn.com/futu-api-doc/ftapi/init.html) Go API【非官方】

![go-futu-api](docs/gopher-niuniu.jpg)

## 在树莓派(arm64)上运行OpenD

> 也可以安装Windows/MacOS/Linux（amd64)等版本，开发时推荐使用GUI版本方便调试

使用[box64](https://github.com/ptitSeb/box64)

先运行`box64 ./FTUpdate`保证软件是最新的, 然后运行`box64 ./FutuOpenD`


## 代码示例

更多代码参考[cmd/main.go](cmd/main.go)

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/qtopie/gofutuapi"
	"github.com/qtopie/gofutuapi/gen/common/getuserinfo"
	"google.golang.org/protobuf/proto"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

  // 建立连接
	conn, err := gofutuapi.Open(ctx, gofutuapi.FutuApiOption{
		Address: "localhost:11111",
		Timeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close() // Ensure the connection is closed when done

  // 构造获取用户基本信息请求
	flag := int32(getuserinfo.UserInfoField_UserInfoField_Basic)
	req := getuserinfo.Request{
		C2S: &getuserinfo.C2S{
			Flag: &flag,
		},
	}
  // 发送请求数据包
	conn.SendProto(1005, &req)
  // 读取响应
	reply, err := conn.NextReplyPacket()
	if err != nil {
		log.Println(err)
	} else {
		var resp getuserinfo.Response
		err = proto.Unmarshal(reply.Payload, &resp)
		if err != nil {
			panic(err)
		}
     // 打印结果
		log.Println(resp.String())
	}

	<-ctx.Done()
	fmt.Println("Main goroutine exiting.")
}
```
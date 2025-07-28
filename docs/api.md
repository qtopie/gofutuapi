## 怎么初始化连接

参考 com.futu.openapi.NetManager#connect

初始化包的代码在这里 com.futu.openapi.FTAPI_Conn#sendInitConnect

参考 sendProto(ProtoID.INIT_CONNECT, req);


返回响应处理参考
handleNetEvents 

onRead


NIO相关 https://juejin.cn/post/7120529881229852685


**简易多用户聊天(go)**

功能:\
    - 支持大厅聊天\
    - 支持使用[\help]查询可用命令\
    - 支持使用[\who]查询在线用户\
    - 支持使用[\rename new_name]改名\
    - 支持使用[\to user_name info]给指定user发送info\

启动服务器:
    `./server_main`

启动客户端:
    `./client_main  [-ip server_ip]  [-port server_port]`

---

``` 
rtchat/

    server_main.go  // server
    server/
        server.go
        user.go


    client_main.go  // client
    client/
        client.go
```


---

ref:
> https://www.bilibili.com/video/BV1gf4y1r79E

<div align="center">
  <img src="assets/open-x365.svg" alt="Open X365" width="460">
</div>

# x365-mobile

X365 协议的 gomobile 绑定：把 [x365-core](../x365-core) 的隧道实现暴露给
Android 客户端的 Go 层。

## 构建

```sh
gomobile bind -target=android ./...
```

## License

MIT

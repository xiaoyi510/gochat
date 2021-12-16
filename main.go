package main

func main() {
	server := NewServer("127.0.0.1",8899)
	// 启动服务
	server.Start()
}

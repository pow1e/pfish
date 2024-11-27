package request

import "time"

type GenerateAgentReq struct {
	Rebuild        bool     `json:"rebuild"`          // 是否构建，如果是则指定使用go/garble 去构建 否则使用模板文件去构建
	Platform       string   `json:"arch"`             // windows则为windows linux为linux mac为darwin
	Arch           string   `json:"platform"`         // amd64
	GrpcServerAddr string   `json:"grpc_server_addr"` // grpc服务端口
	Emails         []string `json:"email"`
}

type CreateAgentConfigReq struct {
	OpenFileName string    `json:"open_file_name"`
	Content      []byte    `json:"content"`
	CreateTime   time.Time `json:"create_time"`
}

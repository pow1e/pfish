package request

type GenerateAgentReq struct {
	UseTemplate    bool   `json:"use_template"`     // 是否使用模板 默认为false
	Platform       string `json:"arch"`             // windows则为windows linux为linux mac为darwin
	Arch           string `json:"platform"`         // amd64
	GrpcServerAddr string `json:"grpc_server_addr"` // grpc服务端口
	Email          string `json:"email"`
}

type CreateAgentConfigReq struct {
}

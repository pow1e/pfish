package database

import (
	"errors"
	"fmt"
	"github.com/pow1e/pfish/config"
	"github.com/pow1e/pfish/pkg/model"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	DB *gorm.DB
)

func Init(conf *config.DataBase) {
	// 创建数据库
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=Local",
		conf.User,
		conf.Password,
		conf.Host,
		conf.Port,
	)
	// 连接 MySQL
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		logrus.Fatalf("连接数据库失败: %v", err)
	}

	// 创建数据库
	createDBSQL := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci", conf.Dbname)
	if err := db.Exec(createDBSQL).Error; err != nil {
		logrus.Fatalf("创建数据库失败: %v", err)
	}
	logrus.Println("数据库创建成功")

	// 切换传切好的数据库
	dsn = fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		conf.User,
		conf.Password,
		conf.Host,
		conf.Port,
		conf.Dbname,
	)
	// 打开 MySQL 数据库连接
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		logrus.Fatalf("连接数据库失败: %v", err)
	}

	// 自动迁移模型（创建表）
	err = db.AutoMigrate(&model.User{}, &model.Message{}, &model.AgentConfig{})
	if err != nil {
		logrus.Fatalf("自动迁移失败: %v", err)
	}

	// 将数据库实例赋值给全局变量
	DB = db
}

func FindUserByEmailMD5(md5String string) (*model.User, error) {
	var user *model.User
	if err := DB.Where("md5 = ?", md5String).First(&user).Error; err != nil {
		// 判断err是否为gorm查询出错的error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// md5不存在
			return nil, errors.New("邮箱md5错误/md5不存在")
		} else {
			return nil, errors.New("内部错误")
		}
	}
	return user, nil
}

func FindMessageByUidPage(uid string, offset int, pageSize int) ([]*model.Message, error) {
	var messages []*model.Message
	if err := DB.Offset(offset).Limit(pageSize).Where("uid = ?", uid).Find(&messages).Error; err != nil {
		return nil, errors.New("内部错误")
	}
	return messages, nil
}

func FindAllMessagesByUid(uid string) ([]*model.Message, error) {
	var messages []*model.Message
	if err := DB.Where("uid = ?", uid).Find(&messages).Error; err != nil {
		return nil, errors.New("内部错误")
	}
	return messages, nil
}

func FindAllMessages(uid string) ([]*model.Message, error) {
	var messages []*model.Message
	if err := DB.Find(&messages).Error; err != nil {
		return nil, errors.New("内部错误")
	}
	return messages, nil
}

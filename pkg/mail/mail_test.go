package mail

import (
	"github.com/pow1e/pfish/config"
	"testing"
)

func TestMail(t *testing.T) {
	conf, err := config.Unmarshal("E:\\Language\\Go\\Go_code\\src\\fish\\config.yaml")
	config.Conf = conf
	if err != nil {
		t.Fatal(err)
	}
	t.Log(conf)

	sendMail("878305612@qq.com")

}

package main

import (
	"github.com/TencentBlueKing/beego-runtime/conf"
	"github.com/TencentBlueKing/beego-runtime/runner"
	"github.com/TencentBlueKing/bk-plugin-framework-go/hub"
	"github.com/beego/beego/v2/client/orm"
	v100 "{{cookiecutter.project_name}}/versions/v100"
)

// InitDB DB 连接初始化
func InitDB() {
	orm.RegisterDriver("mysql", orm.DRMySQL)
	orm.RegisterDataBase("default", "mysql", conf.MysqlConAddr())
}

func main() {
	InitDB()
	hub.MustInstall(&v100.Plugin{}, v100.ContextInputs{}, v100.Outputs{}, v100.InputsForm)
	runner.Run()
}

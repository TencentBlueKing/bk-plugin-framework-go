// +heroku install {{cookiecutter.project_name}}
// +heroku goVersion go1.23
module {{cookiecutter.project_name}}

go 1.23.0

require (
	github.com/TencentBlueKing/bk-plugin-framework-go {{cookiecutter.framework_version}}
	github.com/TencentBlueKing/bk-plugin-runtime-go {{cookiecutter.runtime_version}}
	github.com/sirupsen/logrus v1.9.2
)

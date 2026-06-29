// +heroku install {{cookiecutter.project_name}}
// +heroku goVersion go1.23
module {{cookiecutter.project_name}}

go 1.23.0

require (
	github.com/TencentBlueKing/bk-plugin-framework-go {{cookiecutter.framework_version}}
	github.com/TencentBlueKing/bk-plugin-runtime-go {{cookiecutter.runtime_version}}
	github.com/sirupsen/logrus v1.9.2
)

// bk-plugin-runtime-go v0.2.5 uses bk-apigateway-sdks v1.1.4, which expects
// the pre-v1.3 gopkg cache API. Keep this public replace until a runtime tag
// no longer requires it.
replace github.com/TencentBlueKing/gopkg v1.3.0 => github.com/TencentBlueKing/gopkg v1.0.9

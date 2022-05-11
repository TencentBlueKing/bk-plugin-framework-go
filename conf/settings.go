package conf

import (
	"fmt"
	"os"

	"github.com/beego/beego/v2/core/config"
)

const configData string = `
plugin_name = ${BKPAAS_APP_ID}
plugin_secret = ${BKPAAS_APP_SECRET}
environment = ${BKPAAS_ENVIRONMENT}

user_token_key_name = ${USER_TOKEN_KEY_NAME}
plugin_api_debug_username = ${PLUGIN_API_DEBUG_USERNAME}
`

var Settings config.Configer

var pluginName string
var pluginSecret string
var environment string

var userTokenKeyName string
var pluginApiDebugUsername string

func IsDevMode() bool {
	return Settings.DefaultString("environment", "dev") == "dev"
}

func initPluginName() {
	pluginName = Settings.DefaultString("plugin_name", "")
}

func PluginName() string {
	return pluginName
}

func initPluginSecret() {
	pluginSecret = Settings.DefaultString("plugin_secret", "")
}

func PluginSecret() string {
	return pluginSecret
}

func initEnvironment() {
	environment = Settings.DefaultString("environment", "dev")
}

func initUserTokenKeyName() {
	var tokenDefaultKey string
	if IsDevMode() {
		tokenDefaultKey = "bk_token"
	} else {
		tokenDefaultKey = "jwt"
	}
	userTokenKeyName = Settings.DefaultString("user_token_key_name", tokenDefaultKey)
}

func UserTokenKeyName() string {
	return userTokenKeyName
}
func initPluginApiDebugUsername() {
	pluginApiDebugUsername = Settings.DefaultString("plugin_api_debug_username", "")
	if !IsDevMode() {
		pluginApiDebugUsername = ""
	}
}

func PluginApiDebugUsername() string {
	return pluginApiDebugUsername
}

func init() {
	var err error
	Settings, err = config.NewConfigData("ini", []byte(configData))
	if err != nil {
		fmt.Printf("runtime config load error: %v\n", err)
		os.Exit(2)
	}

	initPluginName()
	initPluginSecret()
	initEnvironment()
	initUserTokenKeyName()
	initPluginApiDebugUsername()
}

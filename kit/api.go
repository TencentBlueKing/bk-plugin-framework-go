// TencentBlueKing is pleased to support the open source community by making
// 蓝鲸智云-gopkg available.
// Copyright (C) 2017-2022 THL A29 Limited, a Tencent company. All rights reserved.
// Licensed under the MIT License (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://opensource.org/licenses/MIT
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on
// an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the
// specific language governing permissions and limitations under the License.

// Package kit collect the basic tool for developer to
// develop a bk-plugin.
package kit

import (
	beego "github.com/beego/beego/v2/server/web"

	"encoding/json"

	"github.com/homholueng/bk-plugin-framework-go/conf"
	"github.com/homholueng/bk-plugin-framework-go/constants"
)

type BkUser struct {
	Username string
	Token    string
}

type PluginApiController struct {
	beego.Controller
	User BkUser
}

type PluginApiControllerInterface interface {
	beego.ControllerInterface
}

func (p *PluginApiController) SetUser(username string, token string) {
	p.User = BkUser{
		Username: username,
		Token:    token,
	}
}

func (p *PluginApiController) Prepare() {
	bkUid, err := p.Ctx.Request.Cookie("bk_uid")
	username := ""
	if err != nil {
		username = conf.PluginApiDebugUsername()
	} else {
		username = bkUid.Value
	}

	token := ""
	if conf.IsDevMode() {
		bkToken, err := p.Ctx.Request.Cookie(conf.UserTokenKeyName())
		if err != nil {
			token = ""
		} else {
			token = bkToken.Value
		}
	} else {
		bkToken, ok := p.Ctx.Request.Header["X-Bkapi-Jwt"]
		if !ok {
			token = ""
		} else {
			token = bkToken[0]
		}
	}
	p.SetUser(username, token)
}

func (p *PluginApiController) GetBkapiAuthorizationInfo(apiPlatform constants.ApiPlatform) string {
	authInfo := F{
		"bk_app_code":           conf.PluginName(),
		"bk_app_secret":         conf.PluginSecret(),
		conf.UserTokenKeyName(): p.User.Token,
	}
	if apiPlatform == constants.ESB && !conf.IsDevMode() {
		authInfo["access_token"] = "access_token"
	}
	b, _ := json.Marshal(authInfo)
	return string(b)
}

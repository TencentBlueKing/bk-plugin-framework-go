// TencentBlueKing is pleased to support the open source community by making
// 蓝鲸智云-gopkg available.
// Copyright (C) 2017-2022 THL A29 Limited, a Tencent company. All rights reserved.
// Licensed under the MIT License (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://opensource.org/licenses/MIT
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on
// an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the
// specific language governing permissions and limitations under the License.

package protocol

import (
	"github.com/TencentBlueKing/bk-plugin-framework-go/hub"
	"github.com/TencentBlueKing/bk-plugin-framework-go/info"
)

// MetaOptions stores runtime-provided metadata for the plugin service meta API.
type MetaOptions struct {
	Code           string
	Description    string
	Language       string
	RuntimeVersion string
	AllowScope     hub.AllowScope
}

// MetaData is the data payload returned by the plugin service meta API.
type MetaData struct {
	Code             string         `json:"code"`
	Description      string         `json:"description"`
	Versions         []string       `json:"versions"`
	Language         string         `json:"language"`
	FrameworkVersion string         `json:"framework_version"`
	RuntimeVersion   string         `json:"runtime_version"`
	AllowScope       hub.AllowScope `json:"allow_scope"`
}

// BuildMeta builds the standard plugin service meta payload.
func BuildMeta(opts MetaOptions) MetaData {
	allowScope := opts.AllowScope
	if allowScope == nil {
		allowScope = hub.AllowScope{}
	}
	language := opts.Language
	if language == "" {
		language = "go"
	}
	return MetaData{
		Code:             opts.Code,
		Description:      opts.Description,
		Versions:         hub.GetPluginVersions(),
		Language:         language,
		FrameworkVersion: info.Version(),
		RuntimeVersion:   opts.RuntimeVersion,
		AllowScope:       allowScope,
	}
}

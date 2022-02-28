// TencentBlueKing is pleased to support the open source community by making
// 蓝鲸智云-gopkg available.
// Copyright (C) 2017-2022 THL A29 Limited, a Tencent company. All rights reserved.
// Licensed under the MIT License (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://opensource.org/licenses/MIT
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on
// an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the
// specific language governing permissions and limitations under the License.

package kit

// Plugin is the interface that wraps all method of a bk-plugin.
//
// Version returns the version number of this plugin.
//
// Desc returns description of this plugin.
//
// Execute define the execution logic with execution context.
type Plugin interface {
	Version() string
	Desc() string
	Execute(*Context) error
}

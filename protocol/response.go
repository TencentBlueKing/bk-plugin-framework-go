// TencentBlueKing is pleased to support the open source community by making
// 蓝鲸智云-gopkg available.
// Copyright (C) 2017-2022 THL A29 Limited, a Tencent company. All rights reserved.
// Licensed under the MIT License (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://opensource.org/licenses/MIT
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on
// an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the
// specific language governing permissions and limitations under the License.

package protocol

// Response is the standard response envelope exposed by plugin service APIs.
type Response struct {
	Result  bool        `json:"result"`
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// OK wraps data in the standard success envelope.
func OK(data interface{}) Response {
	return Response{Result: true, Code: 0, Message: "success", Data: data}
}

// Error wraps an error in the standard failure envelope.
func Error(code int, message string) Response {
	return Response{Result: false, Code: code, Message: message, Data: nil}
}

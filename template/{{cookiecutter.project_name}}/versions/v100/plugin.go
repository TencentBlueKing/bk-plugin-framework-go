package v100

import (
	_ "embed" // import go:embed
	"fmt"
	"log"

	"github.com/TencentBlueKing/bk-plugin-framework-go/constants"
	"github.com/TencentBlueKing/bk-plugin-framework-go/kit"
)

//go:embed form.json
var InputsForm []byte

// Inputs 定义插件的输入
type Inputs struct {
	TemplateID int    `json:"template_id" jsonschema:"title=模板 ID"`
	TaskName   string `json:"task_name" jsonschema:"title=任务名"`
}

// ContextInputs 定义插件的上下文输入
type ContextInputs struct {
	BKBizID string `json:"bk_biz_id" jsonschema:"title=蓝鲸 CMDB ID"`
}

// Outputs 定义插件的输出
type Outputs struct {
	TaskID  int    `json:"task_id" jsonschema:"title=任务 ID"`
	TaskURL string `json:"task_url" jsonschema:"title=任务 URL"`
}

// Store 定义插件的输入
type Store struct {
	TaskID  int
	TaskURL string
}

// Plugin 定义插件
type Plugin struct{}

// Version 定义插件的版本
func (p *Plugin) Version() string {
	return "1.0.0"
}

// Desc 定义插件的描述
func (p *Plugin) Desc() string {
	return "{{cookiecutter.plugin_desc}}"
}

// Execute 定义插件的执行
func (p *Plugin) Execute(c *kit.Context) error {
	state := c.State()
	switch state {
	case constants.StateEmpty:
		var inputs Inputs
		if err := c.ReadInputs(&inputs); err != nil {
			return err
		}

		var contextInputs ContextInputs
		if err := c.ReadContextInputs(&contextInputs); err != nil {
			return err
		}
		log.Printf("create sops task with %v and %v\n", inputs, contextInputs)

		// request with inputs
		taskID := 123
		taskURL := "task_url"
		outputs := Outputs{
			TaskID:  taskID,
			TaskURL: taskURL,
		}
		if err := c.WriteOutputs(&outputs); err != nil {
			return err
		}

		store := Store{TaskID: taskID, TaskURL: taskURL}
		if err := c.Write(&store); err != nil {
			return nil
		}

		c.WaitPoll(5)

		return nil
	case constants.StatePoll:
		var store Store
		if err := c.Read(&store); err != nil {
			return nil
		}

		// fetch task state
		taskState := "RUNNING"
		if c.InvokeCount() >= 5 {
			taskState = "FINISHED"
		}

		switch taskState {
		case "RUNNING":
			c.WaitPoll(5)
		case "FAILED":
			return fmt.Errorf("task %v execute fail", store.TaskID)
		}
		return nil
	}
	return fmt.Errorf("invalid state %v", state)
}

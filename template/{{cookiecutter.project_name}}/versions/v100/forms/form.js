(function () {
  $.atoms["{{cookiecutter.project_name}}"] = [
    {
      tag_code: "hello",
      type: "input",
      attrs: {
        name: "Hello",
        placeholder: "请输入内容",
        hookable: true,
        default: "",
        validation: [
          {
            type: "required",
          },
        ],
      },
    },
  ];
})();

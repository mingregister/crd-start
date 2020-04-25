# crd-start
a crd demo for learning kubernetes source code.

build是打包相关脚本

核心代码都在cmd目录下

pkg/apis目录是我们对自定义资源的定义

pkg/clients目录是k8s的代码生成工具,根据我们的自定义资源自动生成的操作库。
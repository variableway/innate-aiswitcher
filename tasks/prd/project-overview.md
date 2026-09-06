# Project Overview

这是一个Golang CLI 命令行工具，用来进行不同的AI Provider 之间的切换工作的。

当前项目目前期望做一些简化，简化方向是：
1. server 端命令只需要启动server就可以了，不需要太多额外的东西
2. cli端，需要简化命令，不需要太多不太相关的内容，只需要满足provider的list，add，选择就可以了
3. Agent的Profile需要保留，但是就是需要和Provider是联动的，Provider配置好了，直接应该对应的Agent Profile也可以使用

## Task 1: 去除不必要的命令

1. 去除init命令，似乎不是很需要
2. aisw provider 命令 list和preset可以合并，list的时候把provider的templates也一起retrieve出来
3. aisw profile 似乎有重复的ai profile，请检查并且给出对这个建议的看法
4. http://127.0.0.1:8090/, 这个页面需要是light 的theme
5. 运行完aisw命令之后，除非是选择了start agent session这个，其他阅读信息之后都可以返回
6. 修改之后需要更新当前的文档，按照目前命令分别一个命令一个文档，写入到docs目录
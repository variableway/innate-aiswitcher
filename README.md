# README

想要做一个管理ai LLM 的工具，主要负责启动不同的Terminal 命令或者AI Agent的时候，可以随时切换不同的LLM Provider,主要的功能为:
1. 比如Claude-code，在终端一个session启动claude code，可以执行使用minimax，xiaomi，glim 还是默认的的LLM Provider
2. 如果Codex，Claude Code Desktop打开的时候，也可以先使用这个命令行工具切换LLM Provider，然后这些启动的时候自动就使用配置好的LLM Provider了
3. 这些数据可以保存到sqlite，也在本地有配置文件，可以手动修改配置文件，命令行更新同时api key的时候SQLite会更新，配置文件也更新，标准是SQLite数据库，可以使用命令行把SQLite的数据恢复出来
4. 使用golang，TUI形式来编写这个命令行工具，同时可以参考其他当前目录里面的其他项目,使用cc-skills-golang的golang-cli来实现
5. 同样如果UI可以展现当前的信息也是非常好，可以参考其他项目中的UI，展示这些配置和状态信息
6. 

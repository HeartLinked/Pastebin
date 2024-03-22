# Pastebin
A simple backend project in go programing language.

### 技术栈

- Go
- Gin
- MongoDB

### feature

- 路由操作 MongoDB， 使用 Restful API
- 支持用户上传文件，对文件后缀名、文件大小进行校验
- 支持用户上传代码，对代码语言类别进行校验，支持代码高亮
- 基于 MongoDB TTL 接口实现的文件定时自动删除
- 自动分配路由、session实现登录后一定时间无需再次校验

### author

- https://github.com/HeartLinked

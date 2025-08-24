# BaseSQL CLI 使用指南

BaseSQL CLI 是一个命令行工具，让你可以使用类似 SQL 的语法来操作飞书多维表格。

## 安装

### 从源码构建

```bash
# 克隆项目
git clone https://github.com/ag9920/basesql.git
cd basesql

# 构建
make build

# 安装到系统路径（可选）
make install
```

### 直接运行

```bash
# 构建后直接运行
./bin/basesql --help
```

## 配置

### 演示模式

如果你想快速体验 CLI 工具的功能，可以使用演示模式：

```bash
# 使用演示模式（无需真实的飞书应用配置）
./bin/basesql query "show tables" --app-id=demo --app-secret=demo --app-token=demo

# 演示模式下的交互式 shell
./bin/basesql shell --app-id=demo --app-secret=demo --app-token=demo
```

### 生产环境配置

在生产环境中使用时，需要先配置真实的飞书应用信息：

### 1. 初始化配置文件

```bash
basesql config init
```

这会在 `~/.basesql/config.env` 创建配置文件。

### 2. 编辑配置文件

编辑 `~/.basesql/config.env` 文件，填入你的飞书应用信息：

```env
FEISHU_APP_ID=your_app_id
FEISHU_APP_SECRET=your_app_secret
FEISHU_APP_TOKEN=your_app_token
DEBUG_MODE=false
```

### 3. 或使用命令行参数

```bash
basesql --app-id "your_app_id" --app-secret "your_secret" --app-token "your_token" connect
```

### 4. 或使用环境变量

```bash
export FEISHU_APP_ID=your_app_id
export FEISHU_APP_SECRET=your_app_secret
export FEISHU_APP_TOKEN=your_app_token
```

## 基本用法

### 测试连接

```bash
# 测试连接（演示模式）
./bin/basesql connect --app-id=demo --app-secret=demo --app-token=demo

# 测试连接（生产环境）
basesql connect
```

### 执行查询

```bash
# 演示模式示例
./bin/basesql query "SHOW TABLES" --app-id=demo --app-secret=demo --app-token=demo
./bin/basesql query "SELECT * FROM users" --app-id=demo --app-secret=demo --app-token=demo
./bin/basesql query "DESCRIBE users" --app-id=demo --app-secret=demo --app-token=demo

# 生产环境示例
basesql query "SELECT * FROM users"
```

### 执行修改操作

```bash
basesql exec "INSERT INTO users (name, email) VALUES ('张三', 'zhangsan@example.com')"
```

### 交互式模式

```bash
# 启动交互式 shell（演示模式）
./bin/basesql shell --app-id=demo --app-secret=demo --app-token=demo

# 启动交互式 shell（生产环境）
basesql shell
```

#### 🎉 增强功能

交互式 shell 现在支持以下高级功能：

- **📚 命令历史**: 使用 ↑ 和 ↓ 箭头键浏览命令历史
- **🔄 历史持久化**: 命令历史自动保存到 `~/.basesql_history`，重启后仍可用
- **⚡ 自动补全**: 按 Tab 键自动补全 SQL 关键字和命令
- **🚪 多种退出方式**: 支持 `\q`, `quit`, `exit`, Ctrl+C, Ctrl+D

#### 使用示例

进入交互式模式后，你可以：

```sql
basesql> SHOW TABLES;
basesql> SELECT * FROM users;
basesql> DESC users;
basesql> INSERT INTO users (name, email) VALUES ('李四', 'lisi@example.com');
basesql> UPDATE users SET email = 'new@example.com' WHERE name = '张三';
basesql> DELETE FROM users WHERE name = '李四';
basesql> \q  # 退出
```

#### 快捷操作

- **命令历史**: 按 ↑ 键回到上一个命令，按 ↓ 键前进到下一个命令
- **自动补全**: 输入 `SE` + Tab → `SELECT`，输入 `SHOW ` + Tab → 显示补全选项
- **快速退出**: `\q` 或 `quit` 或 `exit` 或 Ctrl+C

## 命令参考

### 全局选项

- `--app-id`: 飞书应用 ID
- `--app-secret`: 飞书应用密钥
- `--app-token`: 多维表格 App Token
- `--config`: 配置文件路径
- `--debug`: 启用调试模式

### 子命令

#### `connect`
测试数据库连接

```bash
basesql connect
```

#### `query [SQL]`
执行 SELECT 查询

```bash
basesql query "SELECT * FROM users WHERE age > 18"
```

#### `exec [SQL]`
执行 INSERT、UPDATE、DELETE 等操作

```bash
basesql exec "INSERT INTO users (name, age) VALUES ('王五', 25)"
```

#### `shell`
启动交互式 SQL shell

```bash
basesql shell
```

#### `config`
配置管理

```bash
# 初始化配置文件
basesql config init

# 显示当前配置（敏感信息会被遮盖）
basesql config show
```

## SQL 语法支持

### 当前支持的操作

⚠️ **注意**: 当前版本的 SQL 解析功能正在开发中，暂时支持基本的 GORM 操作。

### 计划支持的 SQL 语法

- `SELECT`: 查询数据
- `INSERT`: 插入数据
- `UPDATE`: 更新数据
- `DELETE`: 删除数据
- `CREATE TABLE`: 创建表（对应多维表格的数据表）
- `ALTER TABLE`: 修改表结构
- `DROP TABLE`: 删除表

### 示例 SQL 语句

```sql
-- 查询所有用户
SELECT * FROM users;

-- 条件查询
SELECT name, email FROM users WHERE age > 18;

-- 插入数据
INSERT INTO users (name, email, age) VALUES ('张三', 'zhangsan@example.com', 25);

-- 更新数据
UPDATE users SET email = 'new@example.com' WHERE name = '张三';

-- 删除数据
DELETE FROM users WHERE age < 18;
```

## 故障排除

### 连接失败

1. 检查飞书应用配置是否正确
2. 确认应用有多维表格的读写权限
3. 验证 App Token 是否有效

### 配置问题

```bash
# 查看当前配置
basesql config show

# 重新初始化配置
basesql config init
```

### 调试模式

启用调试模式查看详细信息：

```bash
basesql --debug connect
```

## 开发

### 构建

```bash
make build
```

### 测试

```bash
make test
```

### 清理

```bash
make clean
```

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License
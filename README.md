# 120bid 自动化采集工具

120bid 是一个用于自动化访问 120bid.com 网站接口的项目，支持验证码识别、数据抓取与结果处理。
项目主要由 Python 与 Golang 两部分组成，前者负责验证码识别与爬取逻辑，后者负责业务数据处理与存储。

## 功能特性

- 验证码自动识别：基于 ddddocr 模型实现，支持通用字符验证码识别。
- 代理 IP 自动切换：支持青果网络代理池，保障高并发与防封。
- 智能数据抓取：可根据关键词、日期范围、公告类型进行灵活筛选。
- MySQL 数据存储：抓取结果自动入库，方便后续分析与复用。
- 可扩展架构：Python 处理爬虫逻辑，Golang 提供并发与稳定支持。

## 环境配置

### Python 环境

#### 最低要求：

- Python 3.7+
- 建议使用 Python 3.9 或更高
- 支持平台：Windows / macOS / Linux

#### 安装依赖：

进入项目根目录后执行：

```bash
pip install --upgrade pip
pip install -r requirements.txt
```

#### 验证 OCR 安装是否成功

运行以下命令测试 OCR 是否可用：

```bash
python captcha_ocr.py example.jpeg
```

正确输出示例：

```bash
{"text": "9Byx4"}
```

表示环境配置正确。

### Golang 环境

#### 最低要求：

- Golang 1.24+
- 支持平台：Windows / macOS / Linux

#### 编译项目：

```bash
go build
```

生成可执行文件后，直接运行即可。

## 配置文件

配置文件为 config.yaml（位于项目根目录）。

```yaml
# 120bid 配置文件

# 数据库配置
MySQL:
  User: "root" # 用户名
  Password: "123456" # 密码
  Host: "localhost"
  Port: "3306"
  DB: "test" # 数据库名

# 查询参数配置
Params:
  Keywords: [ "超声" ] # 查询关键字
  Status: [ "招标预告","招标公告" ] # 公告类型
  StartDate: "" # 开始日期
  EndDate: "" # 结束日期

# 代理IP配置
ProxyIP:
  AuthKey: "*******" # 青果网络代理IP密钥
  Password: "*******" # 青果网络代理IP密码
```

### 参数说明

| 参数名       | 必选 | 类型       | 说明                  |
|-----------|----|----------|---------------------|
| Keywords  | 是  | string[] | 查询关键字               |
| StartDate | 否  | string   | 开始日期（格式：YYYY-MM-DD） |
| EndDate   | 否  | string   | 结束日期（格式：YYYY-MM-DD） |
| Status    | 否  | string[] | 公告类型                |

## 公告类型参考

| 公告类型     |
|----------|
| 招标预告     |
| 招标公告     |
| 招标变更     |
| 招标结果     |
| 合同验收     |
| 招标信用     |
| 招标预告-意向  |
| 招标预告-需求  |
| 招标预告-意见  |
| 招标预告-预告  |
| 招标公告-重招  |
| 招标公告-第N次 |
| 招标变更-变更  |
| 招标变更-补充  |
| 招标变更-澄清  |
| 招标变更-延期  |
| 招标变更-终止  |
| 招标结果-结果  |
| 招标结果-中标  |
| 招标结果-成交  |
| 招标结果-废标  |
| 招标结果-流标  |
| 合同验收-合同  |
| 合同验收-验收  |
| 招标信用-违规  |
| 招标信用-违约  |
| 招标信用-处罚  |

### 代理IP支持

使用 **青果网络** 提供的高质量代理 IP，支持高并发访问

- 官网地址：https://www.qg.net/?aff=323205
- 在 config.yaml 中填写 AuthKey 与 Password 即可启用。

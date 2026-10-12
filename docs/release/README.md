# Learn-With-AI 安装与使用

Windows 下载与 CPU 匹配的 MSI，双击完成安装，从桌面或开始菜单打开独立的 Learn-With-AI 应用窗口。安装包包含前端、Go 程序、文档解析器、教学生成提示词和许可证，不需要安装 Go、Node.js、Python 或 Office。

| 系统 | 分发文件 | 安装与打开 |
| --- | --- | --- |
| Windows Intel / AMD 64 位 | learn-with-ai-0.7.6-windows-amd64.msi | 双击 MSI，按当前用户安装；从图标打开原生应用窗口 |
| Windows ARM64 | learn-with-ai-0.7.6-windows-arm64.msi | 同上，需要 Windows ARM64 |
| macOS | darwin-…-app.tar.gz / 原生 CI 生成的 .pkg | 放进 Applications 后打开；当前 macOS 版本仍使用默认浏览器界面 |
| Debian / Ubuntu | linux-….deb | sudo apt install ./文件名.deb；当前 Linux 版本仍使用默认浏览器界面 |
| 其他 Linux | linux-…-install.tar.gz | 解压后运行 ./install.sh，从应用菜单打开；界面同上 |

## Windows 安装、覆盖与卸载

MSI 会检测已安装版本。新版升级会显示版本与覆盖确认；取消时不会开始替换。重复打开同版本 MSI 会进入维护界面，可以修复或卸载；不允许降级覆盖较新版本。桌面和开始菜单的 Learn-With-AI 入口直接指向原生应用 exe，使用统一的蓝色书页与电路图标。

旧版自定义 EXE 安装会单独显示迁移确认。确认后先卸载旧应用文件，再安装 MSI 版本；学习数据保留。这个首次 EXE → MSI 迁移是单向的，新安装若失败需重新运行 MSI。后续 MSI → MSI 升级由 Windows Installer 提供事务回滚。升级前请关闭旧应用。

Windows 程序固定安装到 %LOCALAPPDATA%/Programs/learn-with-ai，安装向导不提供目录选择，命令行 INSTALLFOLDER 也无法覆盖此目录。旧版曾安装到其他目录的应用升级后迁入固定目录；旧目录中的非程序文件保留。默认学习数据目录仍为 %LOCALAPPDATA%/learn-with-ai，与程序分离。可从系统「已安装的应用」或 MSI 维护界面卸载，学习项目、笔记、资料和 API 配置保留。

Windows 应用使用 Microsoft Edge WebView2 Runtime。MSI 会检查该组件；若未安装，会显示明确提示。可从微软 [WebView2 官方页面](https://developer.microsoft.com/microsoft-edge/webview2/) 安装 Evergreen Runtime 后重试。缺少组件时不会转而打开普通浏览器冒充桌面应用。

## 应用和资料

Windows 在独立 WebView2 窗口中运行，Go 服务位于同一进程，绑定私有 IPv4 loopback 端口。关闭窗口会隐藏到系统托盘，后台继续运行。单击托盘图标恢复窗口，右键选择“退出”才停止自己的服务。侧栏没有退出按钮。学习偏好和 WebView 用户状态按当前用户保存。

API 与默认模型在侧栏「设置」同页配置，配色在顶部。连接只显示预览，点击打开详情弹窗；提供方预置自动填写地址，可选择模型、填写 Key，自行配置可编辑完整连接参数。复习稿、练习题和学科地图独立调用所选 API，不需要 CLI 助教；练习支持选择题数，答案折叠在每道题下。学科地图可直接生成或根据上传参考资料组织章节。

PDF、Markdown、TXT、PPTX、DOCX、XLSX、CSV 自动本地解析，单文件最大100MiB，上传后自动可供教师查阅。教师按目录、检索和连续原文逐步读取，提供可核对的引用。Office 公式保留结构；PDF 尽量保留位置与数学表达。复杂公式和扫描页可使用设置中单独绑定的视觉模型做有限页识别，可能产生 API 费用；没有绑定时仍可使用本地提取结果。识别结果需要按原页核对，不保证任意教材公式完全恢复。旧 DOC/PPT/XLS 请转换为 DOCX/PPTX/XLSX。

教师、复习稿、练习题和笔记使用统一富文本渲染，支持矩阵、Mermaid 和 SVG；图示可缩放、平移、全屏和导出。交互图形在隔离环境中运行。

已有源码版工作区可通过 --workspace 原工作区绝对路径使用；同一工作区只能由一个应用或开发服务持有，请先关闭旧服务。Windows 的 --no-browser 是明确的无窗口验证模式，不是正常桌面启动方式。可通过 --port 指定开发/验证端口，正常应用自动选择私有端口。

macOS 默认数据为 ~/Library/Application Support/learn-with-ai；Linux 为 $XDG_DATA_HOME/learn-with-ai 或 ~/.local/share/learn-with-ai。这些平台当前保留浏览器运行方式，关闭浏览器不会停止服务，维护者可使用 npm run stop 或 POST /api/system/shutdown 显式停止服务。Linux 用户安装可运行 ~/.local/lib/learn-with-ai/uninstall.sh；Debian 使用 sudo apt remove learn-with-ai；macOS 删除 .app。它们的原生安装验证需对应系统执行，不能从 Windows 交叉构建推断。

## 维护者

仓库 Start-LLL / npm run build / npm start 是快速源码验证工具，保留在仓库中，不进入公开 MSI。开发二进制统一生成到 dist/dev/，发布包位于 dist/releases/，dist/ 根目录不保留 EXE。npm run build:desktop 生成带图标的本机原生开发应用；只有显式执行 npm run desktop:install 才创建原生开发快捷方式，目标是 exe。安装版 MSI 只管理一个正式桌面入口。

使用 node tools/release/build-release.js --version 0.7.6 --installers 在 Windows 构建全部现有目标；Windows MSI 使用固定版本 WiX，构建工具缓存放在 .artifacts/tooling/，不会作为应用运行依赖分发。macOS PKG 仍由 macOS 原生任务生成。Windows 资源嵌入需要 Windows 主机，CI 采用各操作系统自己的目标。

GitHub Actions 在 v 开头的标签或勾选 publish 的手动运行中发布 Release（全部平台冒烟测试通过后才发布）；不勾选时只生成构件。本地包附带 SHA256SUMS 与 THIRD_PARTY_NOTICES.txt。当前构件未进行代码签名。

## 查看已安装版本

在「设置 → 应用 → 已安装的应用」中搜索 Learn-With-AI；或在 PowerShell 中执行：

    Get-ItemProperty HKCU:\Software\learn-with-ai\Installer | Select-Object Version, InstallLocation

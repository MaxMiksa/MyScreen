# 开发与构建

窗口快捷键仅在正式托盘进程启用，复用桌面输入线程。命中含 `Win` 或 `Alt` 的组合键时，回调在拦截主键的同时用 `SendInput` 发送一对无映射键事件，避免系统将修饰键释放识别为单按并打开菜单。回调过滤长按重复，宿主线程负责打开窗口或请求页面保存后关闭。关闭开关时卸载键盘钩子，设置写入失败时恢复原组合键。

## 工程与环境

云彩运动：`sky-preference.ts` 增加每场景 `clouds.motion`，`drift`／`float`，旧数据仅补默认 drift，不清浓度／速度／分布。`cloud-motion.ts` 复用已按速度累计的时钟；随风正向平移，按透视视口半宽＋14 世界单位在完整离屏后循环，轻浮使用不同相位的有界往返。`sky-canvas.tsx` 复用三组／72 粒子、纹理和原有 20 fps 时钟；旧版薄云仅加采样位移，地平线遮罩不移动。无新画布、依赖或硬件动作，0 速／隐藏／初始减少动态效果沿用停帧，停用释放。浏览器四样式、模式／场景记忆／重开／撤销重做、中英最小窗口、0速绘制计数为0及恢复后1.1秒约20次、关闭卸载和初始减少动态效果检查通过；系统减少动态效果运行中切换不会立即停帧，仍需重开窗口，不宣称已修复。数学边界／方向／往返／旧数据兼容检查通过，截图／摘要与正式构建日志在 `build/diagnostics/cloud-motion/`。部署证据 `cloud-motion-20261116-update.json`／`-retention.json`，包 `20261116-175052-276`；时间计划继续，Revision／专注／唯一自启保留、Error／Warning 为空。Go 沿用已提交 overlay 隔离在途原生修改，未重复原生截图或长期资源验收。

部署恢复区分时间计划与手动目标：`Update-NativeGo.ps1` 对原「时间计划」来源使用 resume-plan，核对设备身份、Revision、计划来源、专注／错误及唯一自启，不经手动色温取整或进入手动保持；该分支记录 `PlanResumed=true`，不声称瞬时数值完全不变。手动来源仍逐值恢复并核对。计划路径本轮实际部署通过；回退复用相同恢复逻辑。

点击外部收起：`App.tsx` 只在设置或天气展开时注册 document capture `pointerdown`，沿 composedPath 检查 `data-dismiss-region`；设置根节点、Select Portal 与时间点 Popover 同属 settings，天气根节点属 weather。不拦截／阻止原事件，不监听 pointerup，内部开始的滑条拖动越界不收起；关闭清理监听，沿用设置历史与按需卸载。浏览器验证内部／外部、下拉及嵌套时间点弹层、编辑后重开／撤销、跨界拖动、天气转设置；无浏览器错误。`outside-dismiss-build.log` 记录正式构建／lint、Go test／vet 与 PE 检查，Go 沿用已提交原生 overlay，不夹带工作区显示／恢复／原型页改动。部署 `20261116-172849-082`、唯一自启／专注和五包保留见 `outside-dismiss-20261116-retry-*`；首次部署因渐变中的 5059 K 经手动恢复取整至 5100 K 校验失败并自动回退，重试成功后执行已有 resume-plan 恢复原时间计划，见 `outside-dismiss-resumed-plan.json`。不能把该过程宣称为计划瞬时目标逐值不变；未测本轮原生画面、跨应用外部点击或长期资源。

国内天气：`weather-domestic.ts` 校验 UAPI 实况的所选行政层级、识别到的天气状况、气温及更新时间；缺失或过期才回退 Open-Meteo。缺体感温度、降水量、今日温差或预报不拒绝实况，缺测不补假值；雨况明确而雨量缺测时用基础雨景强度，未来预报只生成建议。缓存沿用 `weather-current-v2`，兼容原有效数据。用 `node --experimental-vm-modules scripts/Test-WeatherSource.mjs` 验证基础实况、缺测雨景、过期及回退；`--live` 另要求深圳南山此刻有合格 UAPI 实况。真实缺字段响应的按时效回放与构建见 `build/diagnostics/weather-core/`。

天气紧凑卡片以 `build/diagnostics/weather-compact/approved.png` 为视觉基准，外框 524×100 DIP、距顶 48；左栏第一行 SVG＋气温、第二行天气＋最低／最高温（不带「今日」），右栏穿衣／出行各一行。`weather-panel-summary` 禁止换行，温差不压缩；过长天气名称或建议单行省略，title 保留完整文字，不能用第三排撑高。24px 圆角、四边 16px 内缩，根网格伸展内框和分隔线，subgrid 共用两行中心；34px／500 气温、18px 天气、16px／400 建议，无新增画布／请求，按需加载、退出卸载不变。通用设置仍可换位置，空位置保留入口。`weather-two-rows/checks.json` 与截图复现用户的晴间多云／26°C／21–27°C，1024 DIP 中英均 100px 高、温差完整、两条建议各 24px 单行；中文天气文字完整，长英文建议可悬停查看。此为隔离测试天气而非实况。正式构建／lint、Go test／vet／PE 检查通过，部署见 `weather-two-rows-20261116-update.json`，保持目标、专注和唯一自启；未重复原生截图／长期内存验收。等距几何验收沿用 `weather-inset/checks.json`，核验内容内框与行中心，不声称字形轮廓逐像素等距。

统一首页：单／双机器背景共用场景栏和天气能力。偏好改名时迁移旧存储键；新键已存在则保留新值。雨景复用画布与粒子上限，切换背景时清理旧水花并重新计算投影。

首页天气卡片：`weather-panel.tsx` 按需加载、退出后卸载，使用本地彩色 SVG，无新增画布、轮询所有者或硬件动作；`App` 管理默认隐藏／Esc 优先级，首页导航保持设置／天气互斥并过滤按住 ↑ 的重复事件，仅未选地区时显示选择入口，其余从通用设置更换地区。卡片与设置共用玻璃材质变量和字体，天气字号按确认图独立还原。`weather-details.ts` 校验所选地点时区的今日最低／最高温及跨午夜未来三小时数据，规则生成建议；请求显式摄氏度／km/h，当前＋两天日界＋四个小时数据由原天气所有者一次读取，未来字段缺失则不给建议，不用全市坐标或旧日期数据凑数。仍兼容旧雨缓存；跨日触发刷新，失败回到正常重试间隔，不热循环；停用雨景且收起卡片时取消请求／清理计时。`build/diagnostics/weather-panel/` 保存确认布局、实时 API 样本、八项规则和浏览器验收、1024 DIP 英文／减少动态效果／旧舞台截图、原生 WebView2＋模拟核心键盘验收及正式构建日志；天气画面使用测试数据，不是南山实况。部署见 `build/diagnostics/weather-panel-20261116-update.json`；原生改动隔离沿用 `weather-location/native-source/source-overlay.json`，未加入其他任务的原生／原型页改动。

天气位置：`frontend/src/lib/weather.ts` 保持原 `WeatherCity` 本地存储结构，搜索结果补全所属城市和层级；无匹配时对中文区县后缀及组合名做有界回退，组合名必须匹配返回行政层级，仍须用户显式选择。缓存键含地点 ID 与经纬度，换区县不复用全市天气；不改天气轮询、桥权限或硬件配置。`build/diagnostics/weather-location/` 保存实时地名响应、六种查询与取消／旧偏好兼容检查、正式前端真实地名搜索／坐标切换／撤销／重做／重载验收和构建日志；天气响应为测试专用晴天，不代表南山实况。正式构建用同目录 `native-source/source-overlay.json` 隔离其他未提交原生改动，未包含原型页；部署见 `build/diagnostics/weather-location-20261116-update.json`。

单一 Go 工程：根目录 `*.go`、`go.mod`、`go.sum` 是 Wails 宿主；`internal/` 是原生适配与纯规则；`frontend/` 是 React/TypeScript 界面；`scripts/` 是构建、开发、部署与诊断入口；`calibrations/` 保存个人设备匹配数据；`build/` 只放可重建产物和诊断记录，不是源码。

```text
React/TypeScript → Wails 桥 → Go 宿主 / dispatcher
                              ├─ win32：DDC/CI、WMI、Gamma、前台会话、颜色恢复
                              ├─ audio：CoreAudio COM
                              ├─ desktop：滚轮钩子、遮罩、反馈、系统浮层
                              ├─ domain：设置、场景、计划、联动与补光规则
                              └─ startup：唯一 HKCU 登录自启项
```

正式构建把 `frontend/dist` 嵌入 exe，无生产 Node/Vite 或业务 HTTP 服务。运行只依赖 Windows x64 和 WebView2；开发依赖 Go 1.27.3、Wails 3.1.0、Node 24.11.1、pnpm 12.16.1。本机 SDK 为 `D:/Apps/go/bin/go.exe`，pnpm 为 `D:/Apps/pnpm/12.16.1/pnpm.exe`，前端锁是 `frontend/pnpm-lock.yaml`。

## 常用命令

```powershell
./scripts/build.ps1 -Check
./scripts/build.ps1 -Check -Package
./scripts/Package-Portable.ps1 -Version v3.0.0
./scripts/Dev.ps1
./scripts/Update-NativeGo.ps1 -Executable <包exe>
./scripts/Measure-NativeMemory.ps1 -Label <唯一标签>
```

- `scripts/build.ps1` 是唯一正式构建入口：冻结安装前端、构建内嵌资源、下载锁定 Go 依赖；`-Check` 追加前端 lint、Go test 与 vet，最终校验 PE 可写段后生成 `build/MyScreen.exe`。`-Package` 另外复制到 `build/packages/时间戳/MyScreen.exe`，时间戳由脚本生成，不在文档写死。构建不部署、不提交、不发布；若正在运行 `build/MyScreen.exe`，须先正常退出再构建；从 `build/packages/` 运行的独立包可继续使用。
- 公开发布先运行 `build.ps1 -Check`，再运行 `Package-Portable.ps1 -Version <vX.Y.Z>`。脚本把正式 x64 exe 和许可证放进 `build/releases/<vX.Y.Z>/MyScreen-windows-x64.zip`。从 ZIP 解压验收后，再将它挂到同一源码提交的 GitHub Release；文件名保持不变，供 README 的「最新版」直链使用。
- `scripts/Dev.ps1` 启动本地 Vite，并以独立 mock 模式打开 `build/MyScreen.exe`；不写硬件、不修改登录自启。启动前完成正式构建；浏览器预览带明确示例数据标识，不能把示例成功当作硬件成功。
- `scripts/Update-NativeGo.ps1 -Executable <包exe>` 读取当前快照，通过认证 IPC 正常退出旧核心，启动新包、等待硬件就绪并恢复目标／专注状态，核对唯一自启项；失败时先排空新核心再尝试独占回退。只传稳定的包路径，不覆盖运行中的 exe；更新结果写入 `build/diagnostics/`。
- `scripts/Measure-NativeMemory.ps1 -Label <唯一标签>` 对当前唯一 Go 核心做五轮面板开关，统计全部自有进程、Private Bytes、专用工作集、CPU 和可见窗口耗时，结果写入 `build/diagnostics/`。标签须唯一；可见窗口不等于渲染完成或可操作时间。
- `scripts/Measure-OwnedProcessMemory.ps1 -OwnerId <PID>` 只读采一帧，按明确的 Go 核心收集后代进程，输出逐进程及合计的独占物理／私有提交和 GPU 归属计数；进程计数不完整时拒绝记录，GPU 计数不可用时明确标记。它不负责切换界面状态；设置是否展开、尺寸、DPI 和场景须由测试流程另行固定。
- `scripts/NativeControl.ps1` 提供 `Send-Go` 函数，连接当前用户、同会话、ACL 限定的本地命名管道，用于运维脚本；命令行 `--call`、诊断与硬件验收入口均不是前端开放动作。

正式 exe 无参数打开面板；`--background` 只启动托盘；唯一登录自启命令为 `MyScreen.exe --mode native --hardware --production --background`。`--production` 要求 `--mode native --hardware`，且不能与开发热加载共存。

## 运行链路与安全边界

- 专注占用仍按副屏全屏／最大化判断。仅对命中副屏的候选窗口检查扩展样式，排除不可激活、鼠标穿透的分层工具窗口（如 Computer Use 光标提示）；Windows 截图选区还须匹配窗口类、样式和进程，普通编辑窗口不豁免。读取身份失败时沿用原占用判断。
- 有副屏遮罩时监听原生桌面层级变化；如果 Computer Use 的跨屏提示被提升到遮罩上方，只恢复本应用遮罩层级，不移动窗体或改动别的进程、焦点、硬件。撤掉遮罩时卸载监听。

- 面板仍按需创建、关闭即销毁。React 提交首页或连接错误后发出 `ui-ready`，宿主只亮出仍有效的窗口；不等待隐藏 WebView 的动画帧或场景素材。建窗时可取消，慢连接约 2 秒兜底。窗口、页面初始底色与舞台一致，不预开 WebView，也不缓存首屏截图。`runtime-ready` 是桥就绪，`panel-shown` 是原生窗口亮出，两者都不是背景图解码完成的时间。
- Wails 前端消息先校验 `http://wails.localhost` 的 Origin/TopOrigin 一致性与消息大小，生产页面通过 CSP 禁止所有 frame；`Backend.Request` 再按动作白名单分发。`dispatcher` 串行处理，仅合并同一控件相邻的同方向调节，快照、保存和其他设备是屏障。桥不提供通用 Shell、文件系统或任意命令执行。
- 本地命名管道只接受当前用户同一物理控制台会话的客户端；`host-memory` 分项诊断和显式 `host-memory-reclaim` 对照只从认证 IPC 进入，不在前端白名单。超时只让调用方继续，不取消仍在执行的硬件／COM 调用。
- 每个 `win32.Worker` 把 DDC 句柄和 COM 引用固定在一个 OS 线程上；`Close` 必须排空队列、等待在途调用结束再释放。同类约束适用于音频 MTA 线程和桌面消息线程。低级钩子回调只做有界的命中判断和入队，硬件、音量与绘制在回调外完成；关闭或退出先撤销钩子，再关闭桌面、音频和显示器。
- 所有亮度、Gamma 和颜色恢复写入前检查物理控制台会话、稳定 PnP 身份与设备映射；编号重新指向其他设备、身份未知或会话切换时拒绝。错误、部分应用和恢复失败分别进入快照警告或错误，不把失败伪装成成功。

## 硬件与颜色所有权

- 已识别外接屏通过 DDC/CI，内屏通过 WMI 写背光；每次变化写合法目标并读回。亮度只作用于实体屏幕，不向后台 RDP 会话发送控制。
- 逐屏 Gamma 以完整 `3×256` 原始表为基线，写入后读回验证；Kelvin 只是近似冷暖标尺，较暖端按 GDI 允许范围整体缩放。CCD 查询失败、复制目标、HDR、高级颜色或目标身份不一致时拒绝写入。
- 只有当前表仍等于本应用最后写入的表时才恢复；检测到未知外部颜色时保留其新值、停止调色并报告冲突。暖色目标约每 3 秒通过串行 worker 核对，仅退回保存的原色基线时重新应用。写入前把原色／最后写入表持久化到颜色 journal；失败保留 journal，重启只在旧属主已结束且身份仍匹配时恢复，不覆盖其他程序新颜色。
- 正常目标是用户亮度，背光目标可能含内屏暖色补光；两者在 `domain.ResolveBrightnessTargets` 中分开。补光仅用于已识别、非主屏、已校准、颜色由本应用持有的内屏；恢复原色、失败或退出前先撤回补光，再恢复颜色。
- 跨屏设备对曲线保存在 `calibrations/` 与用户配置中，按稳定 ID 匹配；缺少配对设备时绝不把曲线套到第三块屏或回退个人数据为默认。补光曲线与跨屏联动曲线独立。

## 配置、场景与启动

- 正式配置：`%LOCALAPPDATA%/MyScreen/native-settings.json`；颜色 journal 为同目录 `native-colors.json`；trace 为 `host-native.jsonl`，同一 Trace ID 贯穿界面接收、排队、硬件工作和回复。 首次启动将旧配置、校准及 WebView2 偏好与缓存复制至新目录，保留旧数据作回退，不复制旧诊断日志。更名兼容期继续与旧程序共用实例锁及硬件互斥锁。
- `domain.Store` 需要显式路径，读取不创建文件；坏文件、未知 schema、非普通文件和权限错误直接返回，不用默认值覆盖。保存比较 `Revision`，仅在成功时递增并返回新副本，使用同目录同步临时文件和原子替换，保留上一版 `.bak`；冲突保留调用方草稿，清理临时文件。锁文件残留须人工核对，不自动夺取。
- 显式迁移只继承已验证的旧版语言、专注延迟和自启偏好；先备份原始字节再首次写入，保留源文件。读取、迁移或核心初始化失败时，绝不按默认设置改动登录自启。
- 场景在 `Targets` 之外保存行为字段；旧场景只在内存补齐，不立即改写磁盘。可选 `ScheduleTransitionMinutes` 仅保存统一时长偏好（包括空计划），编辑时同步各节点，旧配置不迁移覆盖；计划仍按节点时长和本地时钟求值，跨午夜连续；节点行为只应用一次，渐变期间不重复保存或重置专注。无关保存保留手动保持；行为已变化但持久化失败时不能接受返回状态为已提交。
- `startup` 只在配置成功加载／校验后由正式宿主显式调用，维护唯一 `HKCU MyScreen` 项并清理历史别名；操作前快照、逐项读回、失败按相反顺序恢复，保留其他应用条目与已有启用字节。
- 正常退出保留最新正常亮度，卸载输入钩子与遮罩，撤回补光并恢复原色；非正式的原生硬件验收模式恢复到接管前现场；mock 只读写独立模拟配置，不接管硬件。不要让两个会写硬件的核心同时运行。

## 构建 overlay 与升级复核

Go 1.27.3 标准库 `crypto/internal/fips140/drbg` 的固定 32 MiB 可写静态熵源缓冲区会计入 Windows Private Bytes，即使未使用（上游 [issue 81956](https://github.com/golang/go/issues/81956)）。`scripts/New-GoMemoryOverlay.ps1` 生成本项目 `build/go-memory-overlay/`，用线程安全的首次使用分配替换该单一变量；随机算法、系统熵源与已安装 SDK 不变，不声称 FIPS 认证，也没有产品定时 GC 或工作集裁剪。

正式构建必须使用 `scripts/build.ps1` 应用 overlay，并在构建后校验 PE `.data` 可写段小于 8 MiB；直接 `go build` 不会获得该处理。Go 版本、标准库源结构或上游修复发生变化时，`New-GoMemoryOverlay.ps1` 会因精确匹配失败而停止构建；此时先复核是否应移除 overlay，再升级工具链，不能静默跳过。发布前用唯一标签实测内存，不能把历史数字当作新版结论。

并行原生改动隔离：`scripts/build.ps1 -SourceOverlay <json>` 接受 Go overlay 的 `Replace` 映射，合并到内存 overlay；源键必须是仓库内 `.go` 文件，替换文件必须存在，空值表示排除。不修改其他任务工作文件。本次雨景以 `build/rain-native-source/source-overlay.json` 使用已提交 display/recovery 基线并排除未完成测试；替换内容用 `.go.overlay` 后缀，避免 `go test ./...` 把构建目录识别为额外包。

## 前端与界面

- 场景权重由 `scripts/export-scene-region-mask.py` 从对齐的中午原图按实际轮廓离线生成：天空 204、草地 166、机器 141；输出开发母版 `frontend/public/scene-morph-mask-v7.png` 和无损运行图 `frontend/public/scenes/live/mask.webp`。天空不再保留飞絮／星点白斑、横向灰带或扩大保护。天空组件用 `scripts/export-sky-mask.py` 从该权重图派生 9 KiB 的 RGBA 天空遮罩 `sky-mask.png`，CSS cover 与底图对齐，云彩／星星不覆盖草地和机器。遮罩按真实轮廓裁切，无近似地平线或扩大保护。

- `frontend/src/index.css` 是唯一视觉参数入口；全局字号为正文／按钮 16px、小标题 18px、页面标题 26px，胶囊文字行高 1.25，固定面板不随字号或视口改变尺寸；组件复用 shadcn 风格控件、Lucide 图标和 Motion 短反馈。重大界面改动先生成效果图并取得确认，再用与当前源码一致的真实截图核对。
- 快捷面板与设置页共用 1168 × 657 DIP（较上一版宽高再增约 10%；最小宽 1024 DIP） 的 16:9 客户区；`window_ratio_windows.go` 用 `WM_SIZING` 和窗口尺寸事件锁定比例，`main.go` 在首次显示及系统贴靠／最大化后恢复并居中。`resizePanel` 仅保留兼容动作，不再切换窗口尺寸。
- `frontend/public/scenes/` 保存当前版早 / 中 / 傍晚 / 夜四张无文字、无组件的整页 16:9 原创 WebP；`frontend/public/scenes/live/` 保存新版同机位四张 WebP 和一张灰度权重 WebP。仅供开发使用的四张高清 PNG、v7 权重图和 `scene-motion-prototype.html` 留在 `frontend/public/`，Vite 构建结束阶段由 `excludeDevelopmentAssets` 插件从 `frontend/dist` 删除，不进入 exe。正式构建只嵌入场景资源；运行时不请求远程图片或视频，不引入视频解码依赖。真实控件由 React 绘制，避免图片中的按钮与可操作热区错位。
- 新版舞台的 `LiveSceneStage` 默认启用，「设置 → 通用 → 主界面样式」可切回当前版；偏好存于 `localStorage` 并由 `storage` 事件同步，不进入 Go `Settings`、不改 Revision 或配置迁移。底图常态仅一张全尺寸画面、一张权重图和一张 640×320 飞絮天空补片图集；切换时临时多一张目标画面，完成后直接交换并释放旧纹理。再次选场景时，GPU 将当前已显示的基础画面合成为一次性衔接纹理，再直接转向新目标；中断时最多三张场景纹理，完成后恢复一张，不保留历史快照。隐藏、减少动态效果时停止循环；WebGL 初始化失败释放上下文并按当前场景显示原始静态图。
- `SkyEffects` 按当前场景的有效开关创建资源：星星按样式与数量实际分配（上限 18,000），关闭／数量为零／亮度为零即释放；仅创建当前选用的 pmndrs 组件云或旧版薄云，关闭／浓度为零／切换类型时释放旧资源。两类效果均停用时卸载天空画布并释放上下文，恢复时按需重建；延迟图片回调不能复活已释放云彩。保留 `@pmndrs/vanilla` 1.25.0、Three.js 0.187.0、72 云片上限、原纹理／样式／独立闪烁与场景偏好；最高 20 fps，隐藏暂停，减少动态效果启动时绘静态帧。偏好仍复用 `sky-preference.ts` 与自动保存／撤销，天空编辑只预览，不发送硬件场景请求。背景舞台保持独立。

天空按需资源验收：`build/diagnostics/sky-demand-verification.json` 记录 11 项真实浏览器资源检查：稀疏星星仅分配 540 点、密集替换不积累、四种云样式按需切换、零亮度／浓度卸载、反复开关与延迟图片回调安全、场景导航恢复；停用后缓冲／程序释放，天空上下文销毁。减少动态效果仅验证启动时静态绘制，运行中改变系统偏好未验收。正式构建／lint、Go test／vet 和 PE 检查通过，日志为 `sky-demand-build.log`。同场景五次开关各取 15 个样本，中位独占物理内存打开时约 194.32→179.40 MiB，私有提交量 474.45→459.14 MiB；关闭后的 15 个样本均无 WebView2。旧核心长期运行、新核心刚重启，后台独占也从 20.19 降到 13.41 MiB，因此不能把全部降幅归因于天空优化；此为面板打开短测，不等于设置页展开或长期／显存结论。对照见 `sky-demand-memory-comparison.json` 与 before／after 原始记录。

- 飞絮由 `scripts/export-seed-patches.py` 离线制作局部补片，输出 `frontend/public/scenes/live/seed-sky-patches.png` 与 `frontend/src/lib/seed-patches.json`。不改原始 WebP；同一片元程序先补原位置，再从原图减去补片，搬移有符号细节，避免蓝色矩形边缘。10 个偏移量每次绘制计算一次，复用 80 字节数组，不逐像素算轨迹；独立 120 秒无缝时钟，不随场景切换重置。图集 RGB 上传数据 600 KiB，驱动实际内存另测；生成过程只依赖开发机 Python／Pillow／NumPy／OpenCV，产品不依赖它们。
- 快捷面板的四颗按钮按场景名称显式映射早 / 中 / 傍晚 / 夜，不按目标小时就近取场景；无法识别的名称才按亮度兜底。浏览器示例数据与每日计划同步使用四个场景和四个节点。亮度滑条左端月亮、右端太阳，表示从暗到亮；更多控制只列出未映射到四颗按钮的场景。
- 前端通过 Wails runtime 发送请求并等待 `myscreen-reply`；示例数据只在明确 `?preview=1` 的浏览器预览中出现，硬件请求失败不回退成演示成功。
- `@fontsource-variable/noto-sans` 与 `noto-sans-sc` 提供离线可变字体，通过 index.css 的 @import 与全局 --font-sans 应用于文字、表单及图表。Unicode 分片 WOFF2 随 Vite 构建嵌入，只加载所需字形片；不请求远程字体。OFL 完整许可合入项目及分发 notices。
- 品牌图形源为 `frontend/public/favicon.svg`（Lucide Sun–Moon，许可见 `THIRD_PARTY_NOTICES.md`，分发副本嵌入 `frontend/public/third-party-notices.txt`）。更改 SVG 后运行 `D:/Apps/python/python.exe scripts/export-icons.py`，用本机 CairoSVG/Pillow 派生透明底黑／白 ICO；普通构建无需 Python。`scripts/Set-ExecutableIcon.ps1` 在打包前写入 exe 的两组图标资源；`window_icon_windows.go` 按系统深浅色设置窗口大小图标和托盘图标。Windows 的 Wails `App.SetIcon` 实现为空，不能用它代替资源和 `WM_SETICON`。
- `settings-editor.tsx` 是始终挂载的首页玻璃浮层，五个底部分类（当下默认）共享 `lib/settings-history.ts` 的草稿和历史。500 ms 防抖、串行保存、Revision 校验；稳定排序的内容指纹避免 Go/JS 字段顺序引起重复保存。保存期间的新改动继续排队，失败保留草稿；撤销／重做（含本地外观偏好）同样保存。Esc 关闭和首页场景请求先 flush，失败重新展开浮层。状态推送不抢焦点或覆盖未保存内容。
- 窗口为 frameless，无顶部窗口栏；保留顶部 28px 隐形拖动区。设置浮层由胶囊向上裁切展开，无向下箭头，顶部不留操作行；重新检测与「已自动保存」及变化图标位于底栏左侧，浮层固定 958×526 DIP，左右与上下始终居中，不随主窗口缩放；内部布局不依赖视口断点，五导航与收起胶囊共用垂直中心锚点（距浮层底边约 32px），不以各自底边对齐；首页场景栏及按钮均按内容定宽，取消百分比宽度、最小固定宽度与 flex 等分；外框内边距和按钮间距由 --nav-focus-size 按比例派生，内部 scene-capsule-focus 按内容包裹；它与设置页选中按钮共用玻璃材质，高 38px、文字左右 15px 留白，纯图标 38px 正圆；图标 19px／2.1px 描边及 96% 白色、内外框增强边缘对比，外框上下高光强度一致、选中圈不加单侧高光，阴影垂直偏移为零，避免光影破坏等距观感，无背景模糊；历史组右／下等距 14px。专注／通用以独立分组固定归属，场景两列网格纵向延伸、四项同屏。舞台 overflow: clip 防止焦点自动滚动破坏居中。首页方向键／WASD 切场景与展开；浮层内切分类，↓ / S 收起；Esc 收起后再按通过既有 `window-close` 销毁面板并保留托盘核心。输入控件、组合输入和已处理的按键不被全局导航劫持。

联合版验收：`build/diagnostics/sky-weather-unified-20261115.json` 记录五次开关面板；15 个关闭后采样均为 0 个 WebView2，最后关闭的 Private Bytes／专用工作集约 35.30／18.67 MB。仅证明本次开关释放路径，不证明长期无泄漏或已测显存。部署与唯一自启证据见同前缀 `-update.json`，前端／正式构建日志、样式截图和隔离源码清单见 `build/diagnostics/sky-effects/`。

## 开发验收原则

搜索取消状态：每次城市搜索注册一次所属请求的取消监听，只有它仍是当前请求时才立即复位 loading；finally 移除该监听，并仅清理自身请求引用，不能覆盖新请求状态。`search-cancel-verification.json` 验证真实设置形变在 83.8ms 内收起／反向展开、组件未卸载、相同 query 可直接重搜；故意推迟旧请求结束，验证旧 finally 不清掉新 loading，过时结果不覆盖新结果。补丁仅改取消状态与监听清理，不改渲染／资源策略；沿用 `20261116-060555-867` 的既有内存矩阵、13 项资源验收及 20 次循环，不为新包声称重新测量。补丁包的构建／lint、Go test／vet 与 PE 检查见 `search-cancel-production-build.log`；最新不可变包与部署参数仍以 `handoff.json` 为准，未部署／提交／推送。

界面内存资源：`sky-effects.tsx` 仅保留偏好订阅与有效开关判断，首次需要云／星时懒加载 `sky-canvas.tsx`（复用已验证的按需分配逻辑）；Three.js 通过 Vite `dedupe` 保证统一实例，当前锁文件原本就无重复版本，不宣称去重节省。天气与天空编辑模块分别进入才加载；设置内容在形变归零后卸载，外框、导航、草稿／保存和历史保留，支持中途反向与减少动态效果。天气搜索收起时立即取消，未提交搜索文字保存在外层；临时结果随内容卸载释放。背景初始化等待各图片结果后统一清空解码源，包含部分失败与取消；保持原 GPU 纹理、画质和动画，不引入额外位图副本、强制 GC 或工作集裁剪。模块代码加载过后仍由浏览器缓存，卸载组件不等于卸载脚本。

内存证据：`build/diagnostics/memory-demand/` 保存 `matrix.json`／原始逐进程计数、`static-reduced.json`、`resource-verification.json`、`stability.json` 和 `production-build.log`。基线为 `ff25273` 已提交雨景，正式资源在独立模拟核心的原生 WebView2 中测试；1168×657 DIP、DPR 1.5、中午场景，新旧核心按 ABAB 各启动两次，每状态各 6 个样本，计入核心与全部后代进程。独占物理／私有提交分别统计；GPU 数字是进程归属计数，不能当作整机去重后的新增显存。矩阵中的 `static` 指仅背景、原有飞絮仍运行；真正静态由减少动态效果媒体模拟并重载页面后单列，重载缓存使其绝对值不能与其他状态直接相减。模拟核心不含硬件线程，绝对数值不能代替正式硬件版占用。

短测中仅背景独占物理中位约 188.04→173.94 MiB、私有提交约 420.95→411.37 MiB；真正静态背景独占约 210.62→197.97 MiB、私有提交约 505.82→467.54 MiB；天空、雨、组合与设置的物理／提交量变化不一致，私有提交未全面下降，不能宣称各状态／指标同步下降或已证明长期收益。组合效果 90 秒中位采样独占约 185.2→147.2 MiB、私有提交约 503.8→366.2 MiB，未见持续增长；这是短时趋势，不证明长期无泄漏。13 项真实 WebGL 检查覆盖云星／雨各 20 次循环、迟到纹理、搜索取消／草稿、跨页撤销、收展反向、减少动态效果、初始图片部分失败／取消；天空与雨的上下文销毁，Three.js 内部占位纹理由上下文丢失一并失效。关闭后矩阵及稳定性采样均无自有 WebView2。正式构建／lint、Go test／vet 与 PE 检查通过；发布包及构建来源见同目录 `handoff.json`；主 agent 验收后已部署 `build/packages/20261116-133948-719/`，唯一自启与目标／专注保留见 `build/diagnostics/memory-demand-20261116-update.json`。

画布合并未合入：背景为原生 WebGL1，Three.js 0.187.0 自 r163 起仅支持 WebGL2；天空还通过独立 CSS 轮廓遮罩裁切。直接共享上下文不兼容，需迁移背景渲染与天空遮罩并重新验证过渡／颜色。矩阵仅证明天空开启会增加图形资源，不证明合并可省同等容量；目前没有保持相同画面并证实收益的合并原型。保留底层背景／天空、中层控件、上层雨水的层级，不改变已确认的雨丝与外框溅落。

雨景运行时：`frontend/src/lib/wind-rain-runtime.js` 是正式版与浏览器预览共用的唯一斜风雨实现；锁定 three.quarks 0.17.1（MIT）并复用现有 Three.js。`WeatherRainOverlay` 只在可见、非减少动态效果且雨量大于零时懒加载并创建一个透明画布；停雨、隐藏及卸载释放粒子系统、批次、材质、纹理、计时器、观察器与 WebGL 上下文。模块代码可被浏览器缓存，停止下雨不等于卸载模块。DPR 1、30 fps，三张微型纹理；三层最大总速率 1380 滴／秒，雨丝／水珠／短弧上限 2400／144／24。复用水珠与短弧系统，不为每次溅落新建系统。

雨丝连续经过机顶源图 y=378、x=722–945 时产生短促水珠与少量短弧，不改变雨滴轨迹／寿命。真实 `.settings-morph-surface` 与收起提示的上缘按每帧矩形定位，跟随连续形变；雨层位于控件之上且点击穿透。设置展开同步清除并关闭机顶水花，收起恢复；外框溅落保留。氛围由背景内静态渐变实现，不加模糊／纹理／计时器；`rain-appearance.ts` 共用雨量曲线，小雨透明度趋零、最大雨量保持，夜景减轻压暗。

天气请求：`weather.ts` 仅请求固定 HTTPS `uapis.cn`／`api.open-meteo.com`／`geocoding-api.open-meteo.com`，生产 CSP 仅允许这些天气来源；不经过硬件队列、不增加 Go 后台轮询。每次请求 10 秒超时、64 KiB 响应上限、禁凭据／跳转，并校验地区、天气字段／代码／时间；国内来源失败才回退，取消不发起备用请求。15 分钟刷新，30 分钟缓存到期停雨，隐藏取消请求，卸载取消定时器。`weather-settings.tsx` 负责搜索／选择与 30 秒预览，偏好纳入撤销历史；不改变硬件目标。

浏览器选型：`scripts/rain-preview/` 中运行 `npm ci --ignore-scripts`、`npm run build:ui`、`npm start`，访问 `http://127.0.0.1:5194`。六款配置仅用于比较；斜风雨 re-export 正式运行时。默认「实际界面」通过只读模拟桥复用实际设置／胶囊组件，不调用硬件；预览生成到 `build/rain-ui-preview`，不进入正式 exe。`/product/?preview=1` 使用 `frontend/dist` 检查真实正式资源；生产环境不提供此开发服务。零雨量、暂停、开关、视口变化与切换销毁已验证。

雨景验收：正式资源浏览器检查通过设置展开机顶水花归零、收起恢复、边框持续溅落、点击穿透与 30 秒预览到期画布释放，无控制台错误。构建／lint、Go test／vet 与 PE 检查见 `build/diagnostics/rain-production-build.log`。部署证据见 `rain-wind-final-20261116-update.json`（最终包仅补齐依赖许可，构建日志 `rain-production-final-build.log`）；五轮面板开关后 15 个关闭采样均无 WebView2，最后关闭私有提交量约 37.38 MB、专用工作集约 19.62 MB，见 `rain-wind-20261116.json`。此检查使用当前用户天气偏好，仅证明面板释放路径；新粒子雨景的原生雨开／关增量、长期内存及显存尚未测定，旧 shader 雨景短测不能用于新版结论。

优先保证功能，再做与风险相称的最小检查：`scripts/build.ps1 -Check` 覆盖冻结安装、前端 lint／构建、Go test／vet 与正式构建。涉及真实硬件时，先正常退出现用核心并确认颜色与亮度恢复，再独占运行显式验收入口；普通测试和浏览器预览不写真实显示器或音量。

诊断输出放 `build/diagnostics/`；清理 `build/` 前检查现用进程和登录自启，保留它们指向的包。前端使用项目级共享 pnpm store；改变目录层级后重新安装依赖链接，不清共享缓存。保留用户配置、个人校准和共享依赖缓存；清理一次性临时文件。阶段完成后仅做必要验证，部署正式版，再提交并推送本轮改动；不夹带工作区其他未完成改动。已合并／过时 worktree 自动清理，托管 worktree 用归档工具；唯一主工作区保留。成功部署后由 scripts/Remove-OldPackages.ps1 将本地 build/packages 中超过最近 5 个的旧包移入回收站；清理前一次性核对所有目标位于包目录内且非运行／自启路径，未知目录不删除；不清理远端历史。真实设备的目标状态、错误、Warning 和唯一自启项应在每次部署后重新核对。界面准则见[产品说明](PRODUCT.md)，本次验证状态与未测边界见[状态说明](STATUS.md)。

全局下拉菜单复用 `frontend/src/components/ui/select.tsx` 的 Radix UI Select，浮层通过 Portal 避免设置内容区裁切，样式定义在 `index.css`。设置导航忽略 combobox／option 内的方向键，应用 Esc 让位于打开的菜单；组件关闭后恢复触发器焦点。

## 云彩位置分布

首页已撤下临时云朵测试栏与 Takram／体积／卡通候选、对应资源和依赖。保留 pmndrs 三种原有云彩和旧版薄云增强；cloud-legacy.ts 不改。cloud-placement.ts 为三组现有组件计算原有分布、分区散布、错落高度、弧形、远近深度五种位置；同场景／样式使用固定种子，不因刷新重新随机，移动围绕基准位置缓慢漂移。形状、纹理、浓度、速度与72实例上限不改，不增加画布或依赖。默认仍为原有分布。

位置选择仅在非旧版云彩的天空设置中显示，四场景分别记忆，沿用自动保存／撤销。旧偏好缺少layout时回退original；未知layout也回退，不改写读取到的本地存储。旧版薄云与星星不受位置算法影响。

部署包 build/packages/20261116-004217-009；目标／专注保留与唯一自启证据 cloud-placement-20261116-update.json，验证记录及界面截图位于 build/diagnostics/cloud-placement/。

星星闪烁：star-twinkle.ts 仅适配pmndrs星点材质，初始化一次逐星相位／频率／幅度属性；取消默认共享尺寸脉冲，用独立明暗曲线保留星点形状。每帧只更新时间，闪烁滑条控制强度，0和减少动态效果常亮。云彩与旧版薄云不改。设置收起修正：clipPath完成收拢后才淡出，天空编辑内容不因open=false提前换页，关闭期间inert与pointer-events禁止操作。

蒲公英移动范围：`live-scene-stage.tsx` 的水平／竖直振幅均为原来两倍，共享相位周期由 120 秒延长至 240 秒，保持各朵瞬时漂移速度与四场景连续相位，不增加资源或修改其他天空效果。正式构建／lint、Go test／vet、PE 检查见 `build/diagnostics/seed-range-build.log`；部署及保留包证据见 `seed-range-20261116-update.json`／`seed-range-20261116-retention.json`，保持显示目标、专注与唯一自启。未重复原生画面或长期验收。

## 蒲公英随风漂浮

单机器背景的天空设置沿用现有 Select，按场景保存 `seeds.motion=float|wind`，旧／未知值回退原有轻漂，读取不覆写；自动保存、预览与历史复用现有链路。双机器无可分离飞絮素材，隐藏控件且不为它读取风。`ScenePanel` 统一订阅天空偏好并传给云星与舞台，避免重复持有偏好。

`weather-wind.ts` 读取现有当前天气：UAPI 的当前风向与蒲福等级取视觉等效中值（标记 estimated，不冒充实测 km/h，不用未来预报冒充当前，也不冒充阵风）；Open-Meteo 在同一个当前请求增加风速／来向／阵风，显式 km/h。无风、未知方向、无位置／缺失／旧缓存／过期均回退模拟轻风。复用单一天气所有者、15分钟刷新／30分钟失效／隐藏取消；雨景关闭时仅当前启用随风的可见舞台请求天气，不新增后台所有者或补充风请求。字段依据 [UAPI](https://uapis.cn/docs/api-reference/get-misc-weather)、[Open-Meteo](https://open-meteo.com/en/docs)，风级等效参考[气象学会蒲福表](https://www.rmets.org/metmatters/beaufort-wind-scale)。画面将东映射右、北映射上，气象来向反转为去向并压缩纵向，不声称场景相机具有真实罗盘方位。

`seed-motion.ts` 对原10朵做共享缓变风场、逐朵错速／起伏／小角度转动、离屏上风侧错时补入；小雨逐朵淡出，大雨全部渐隐，雨停恢复。沿用原素材图集、灰度机器／草地遮罩、WebGL画布与帧率；粒子状态直接复用原 uniform 数组（20→40个 float，总160B，净增80B），随风模式仅再持有3个标量，不创建逐帧数组、额外纹理／快照／计时器／依赖。隐藏／减少动态效果不推进，原有轻漂的位置公式及速度保留。

最小检查：`node scripts/Test-SeedMotion.mjs` 覆盖偏好兼容、风向／强弱、平滑反向、逐朵转动、雨隐恢复与30分钟模拟边界／补入（不是30分钟真实运行）；`node --experimental-vm-modules scripts/Test-WeatherSource.mjs` 覆盖同请求风字段、来源／取消与原天气规则。`build/diagnostics/seed-wind/` 保存检查、正式构建／lint／Go test／vet／PE日志、实际组件截图与浏览器证据：20次模式切换常态3张舞台纹理、场景中断峰值5、舞台卸载0，重新挂载回到3；小雨透明度总量约6.49／10、大雨约0.000008／10；重载／撤销重做、双背景适用范围及1024 DIP英文布局通过，无最终页面错误。诊断天气为受控样本，非当地实况；未重复原生画面、完整硬件回归或长期CPU／内存／显存测量，不声称整体进程内存零增加。部署 `seed-wind-20261117-update.json`，原计划继续、专注及唯一自启保留；包清理由部署脚本保护现用／自启路径。

内存优化的对照口径：保持场景、画质、帧率和后台功能，分别统计打开与关闭面板的自有进程树；纹理容量和进程占用分开报告。

设置按交互加载：首屏只挂载用于形变测量的空外框，普通入口与天气选址在首次打开时创建编辑器。收拢完成卸载控件；同一窗口保留草稿、撤销历史和保存冲突。

双机器背景按当前选择解码，原图、图层顺序与 900ms CSS 淡入淡出保留。退场结束移除图片节点，不主动清理浏览器共享缓存。

二维特效候选与正式资源隔离，构建及 A/B 口径见[二维效果预览](../scripts/rain-preview/canvas2d/README.md)。普通云星和二维雨不依赖 Three.js；旧薄云回退原实现，单机器实时背景不改。

## 连续场景过渡

设置页／场景栏共用 `settings-morph.tsx` 的空玻璃外框，Motion 阻尼弹簧保留反向时速度；上下文仅共享进度与两端矩形，ResizeObserver 随窗口／栏宽更新锚点。外框位置、尺寸、圆角、材质及阴影随同一进度插值；设置内容不缩放，仅裁切／淡入淡出；场景内容在最后 3.5% 形变中交接。保持设置挂载和草稿，立即切换 inert，不依赖延时定时器；无截图／画布／纹理缓存，卸载释放观察器。系统减少动态效果时直接跳至终态。逐帧证据 `build/diagnostics/settings-morph/frames.json` 覆盖开关、90ms 间隔五次反向、图标／图文胶囊，验证无两组内容重叠、外框有界及文字 transform 为 none；未实测系统减少动态效果与长期占用。

`ScenePanel` 以同步 ref 记录最新选择，按键和点击不依赖上一帧 React 状态；请求序号只允许最新手动请求结束等待，串行硬件中间回报不覆盖新目标，结束后恢复外部场景同步。仅增加常量状态，不增加画布／纹理／历史快照。`build/diagnostics/scene-input-fix/verification.json` 记录同帧正反向三次、两条中间回报与等待结束后的外部同步；浏览器使用真实组件和受控回报，不写硬件。

- `LiveSceneStage` 不再等待进行中或过时的加载结束：新目标取消旧图片加载，加载完成后再次核对最新目标；废弃请求不得清空新请求状态。中断时从最后已显示的进度与飞絮相位衔接，单次 GPU FBO 合成、不 `readPixels` 或 CPU 拷贝。
- 纹理按显示宽度生成且不超过 1672×941，RGBA 快照理论上限约 6.29 MB；中断瞬时最多三张场景纹理，过渡两张、空闲一张，另有共用 mask／飞絮图集，不常驻四场景。雨景后处理不写入快照，飞絮只修补新目标；快照失败保留当前画面。隐藏或减少动态效果时直接切到目标；卸载取消加载并释放纹理与上下文。硬件请求仍串行，取消画面过渡不等于取消在途硬件调用。
- 验收：快速三场景切换、真实 WebGL 晴天／雨天衔接与过时加载均通过，平均 8 位像素差 0.09～0.22，无 GL 错误；峰值 5、空闲 3、销毁 0 张纹理（均含共用 mask／图集）。证据为 `build/diagnostics/scene-interrupt/gpu-check.json`，正式构建日志见同目录 `production-build.log`。未验证长期 CPU／显存走势或所有机器的流畅度。

当前正式包以 STATUS.md 为准；界面内存优化的部署／目标与专注保留／唯一自启证据为 `build/diagnostics/memory-demand-20261116-update.json`，包保留结果见同前缀 retention 文件。发布使用已提交原生基线叠加本轮六个前端文件，未夹带工作区未完成的显示／恢复修改；临时源码和测试环境已回收。

## 渐变交互图

`schedule-chart.tsx` 随首次打开设置加载，避免切入渐变页才等待模块、短暂清空内容；首页仍不加载编辑器，图表及日出日落请求只在渐变页启用。保留单个 SVG 和原采样，缓存当前两条曲线字符串，鼠标悬停或仅显示器当前数值变化不重新采样。圆点与标签放在底部时间轴；Pointer Capture 拖动仅保留当前节点预览，松手一次提交共享历史，取消不保存。离页／收拢后随设置内容卸载；弹层仅打开时挂载，采用 [Radix Popover](https://www.radix-ui.com/primitives/docs/components/popover)，场景选择沿用全局 Select。原生时间输入通过 input 事件同步；Esc 优先关闭弹层，普通节点焦点仍允许收起设置。纯规则 `schedule-edit.ts` 镜像原生跨午夜、重复分钟及下一节点间距约束，冲突拒绝提交，不暗改场景或时长。

`build/diagnostics/schedule-chart/` 保留布局截图、交互／资源观察摘要及正式构建日志；浏览器预览不写硬件，验证添加、精确时间／场景、拖动后一次撤销、重复／重叠拒绝、删除／撤销重做、Esc 优先级、离页无图表／弹层；边界规则10项通过。资源观察为1个SVG、0个图表画布、0个右侧节点行，无浏览器错误，不代表实测内存降低。正式构建用原生已提交基线 bef9385，隔离工作区显示／恢复及原型页的其他修改，临时源码已回收。部署包 `build/packages/20261116-160754-107/`；显示目标／专注、无错误与唯一自启及保留五包证据为 `schedule-chart-20261116-update.json`／`-retention.json`。未重复原生画面、长期内存／显存或硬件计划触发验收。

## 日出／日落节点

`internal/domain/solar.go` 使用 go-sunrise v1.1.0（MIT，许可随包分发）本地计算，按本机日期、坐标和时区缓存固定五天；同日、同计划复用展开事件，不新增后台网络或轮询。`TimeKind`／`OffsetMinutes`／`SolarLocation` 为兼容旧配置的可选字段；原固定计划维持既有校验。混合计划按真实时刻展开、应用偏移、固定／日出／日落与列表顺序择一处理同分钟，渐变仅在求值时裁剪到下一事件，不写回定义。手动保持和实时衔接沿用缓存。极地缺失事件不补造，固定节点仍可执行。

桥新增只读 `solar-times`，只接受已验证地点，不走硬件队列或修改配置；返回前今后三天 RFC3339 事件供偏移跨午夜与夏令时解析。前端仅在渐变页请求，单个午夜计时器及恢复可见补刷新，离页取消计时器与监听、忽略迟到结果；显式 `?preview=1` 演示时刻不参与生产。太阳节点拖动转固定；零偏移与固定类型在保存时规范化为省略值，与Go响应保持一致，避免重复自动保存。

本轮证据：`build/diagnostics/solar-schedule/` 的 `editor-rules.mjs`／`verification.json`／`layout.jpg`／`production-build.log`／`native-solar-preview.json`。前端14项断言、Go五组太阳测试及只读桥测试、正式构建／lint／全Go test／vet与PE检查通过；浏览器检查偏移、拖动／撤销、同分钟可编辑、渐变缩短、城市入口与弹层／离页释放，图表仍1个SVG、0画布，无浏览器错误。截图仅为演示时刻。原生只读深圳计算2026-11-16日出06:37／日落17:41，配置与专注不变。部署包 `build/packages/20261116-200209-039/`，原时间计划继续，唯一自启与Revision／专注保留、无Error／Warning，包保留五个，见同名前缀部署／retention记录。构建仅隔离未完成display／recovery源码与未跟踪测试，开发原型由现有构建规则排除；临时覆盖源回收。不新增时间轴库、画布或天文服务请求，未实测长期内存或跨日硬件触发。

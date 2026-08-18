using System;
using System.IO;
using System.Net;
using System.Net.Sockets;
using System.Windows;
using AppsManager.Services;

namespace AppsManager
{
    public partial class App : Application
    {
        protected override void OnStartup(StartupEventArgs e)
        {
            if (e.Args.Length > 0 && e.Args[0] == "--test")
            {
                RunSelfDiagnostics();
                Shutdown();
                return;
            }
            base.OnStartup(e);
        }

        private void RunSelfDiagnostics()
        {
            string logPath = Path.Combine(AppDomain.CurrentDomain.BaseDirectory, "test_result.log");
            using var sw = new StreamWriter(logPath, false);

            void Log(string msg)
            {
                Console.WriteLine(msg);
                sw.WriteLine(msg);
            }

            Log($"[{DateTime.Now}] 开始自动化诊断自测试...");

            try
            {
                // 测试 1：端口检测测试
                Log("测试 1：本地端口监听与 PID 检测...");
                int testPort = 9999;
                var listener = new TcpListener(IPAddress.Loopback, testPort);
                listener.Start();

                bool inUse = PortService.IsPortInUse(testPort);
                var pids = PortService.GetPidsByPort(testPort);

                listener.Stop();

                Log($"  - 端口被占用检测结果 (预期 True): {inUse}");
                Log($"  - 占用该端口的 PID 数量 (预期 >= 1): {pids.Count}");
                if (pids.Count > 0)
                {
                    Log($"  - 检测到的 PID: {string.Join(", ", pids)}");
                }

                if (!inUse || pids.Count == 0)
                {
                    throw new Exception("端口检测逻辑失败：未检测到占用的端口或 PID。");
                }
                Log("测试 1 通过了！");

                // 测试 2：智能项目探测测试
                Log("测试 2：智能项目文件识别探测...");
                string tempDir = Path.Combine(AppDomain.CurrentDomain.BaseDirectory, "test-temp-project");
                if (Directory.Exists(tempDir)) Directory.Delete(tempDir, true);
                Directory.CreateDirectory(tempDir);

                string pkgJson = "{\n  \"name\": \"test-auto-node\",\n  \"scripts\": {\n    \"dev\": \"vite\"\n  }\n}";
                File.WriteAllText(Path.Combine(tempDir, "package.json"), pkgJson);

                var detected = ProjectDetector.DetectProject(tempDir);
                Directory.Delete(tempDir, true);

                if (detected.SubServices.Count != 1)
                {
                    throw new Exception($"预期有 1 个子服务，实际得到 {detected.SubServices.Count} 个。");
                }

                var sub = detected.SubServices[0];
                Log($"  - 识别出的服务名 (预期 test-auto-node): {sub.Name}");
                Log($"  - 识别出的启动命令 (预期 npm run dev): {sub.StartCommand}");
                Log($"  - 识别出的默认端口 (预期 5173): {sub.Port}");

                if (sub.Name != "test-auto-node" || sub.StartCommand != "npm run dev" || sub.Port != 5173)
                {
                    throw new Exception("智能识别探测逻辑不匹配预设期望。");
                }
                Log("测试 2 通过了！");

                // 测试 3：命令行粘贴解析测试
                Log("测试 3：命令行粘贴提取解析器...");
                string testCmd = "cd web && pnpm dev:ip --port 5555";
                string baseRoot = AppDomain.CurrentDomain.BaseDirectory.TrimEnd(Path.DirectorySeparatorChar);
                var parsed = ProjectDetector.ParseCommand(testCmd, baseRoot);

                Log($"  - 解析出的服务名 (预期 web): {parsed.Name}");
                Log($"  - 解析出的命令 (预期 pnpm dev:ip --port 5555): {parsed.StartCommand}");
                Log($"  - 解析出的端口 (预期 5555): {parsed.Port}");
                Log($"  - 解析出的工作目录 (预期以 \\web 结尾): {parsed.Path}");

                if (parsed.Name != "web" || parsed.StartCommand != "pnpm dev:ip --port 5555" || parsed.Port != 5555 || !parsed.Path.EndsWith("web"))
                {
                    throw new Exception("复合命令智能解析逻辑与期望结果不符。");
                }
                Log("测试 3 通过了！");

                // 测试 4：ANSI 颜色文本解析自测试
                Log("测试 4：ANSI 颜色转义字符解析渲染器...");
                var testPara = new System.Windows.Documents.Paragraph();
                string ansiText = "\x1b[32m[VITE]\x1b[0m \x1b[31mError occurred\x1b[0m";
                AnsiColorParser.ParseAndAppend(testPara, ansiText);

                Log($"  - 解析后的 Run 节点数量 (预期 3): {testPara.Inlines.Count}");
                if (testPara.Inlines.Count != 3)
                {
                    throw new Exception($"ANSI 解析输出节点数量异常，得到 {testPara.Inlines.Count} 个。");
                }

                var r1 = (System.Windows.Documents.Run)testPara.Inlines.FirstInline;
                var r2 = (System.Windows.Documents.Run)testPara.Inlines.FirstInline.NextInline;
                Log($"  - 节点 1 文本 (预期 [VITE]): {r1.Text}, 颜色 (预期 Green): {r1.Foreground}");
                Log($"  - 节点 2 文本 (预期  ): '{r2.Text}', 颜色 (预期 LightGray): {r2.Foreground}");

                if (r1.Text != "[VITE]" || r1.Foreground != System.Windows.Media.Brushes.Green)
                {
                    throw new Exception("ANSI 颜色或文本断言匹配失败。");
                }
                Log("测试 4 通过了！");

                // 测试 5：智能端口日志解析纠偏测试
                Log("测试 5：日志智能端口提取捕获器...");
                string viteLog = "  - Local:   http://localhost:5666/";
                string coloredViteLog = "\x1b[32m➜\x1b[0m  \x1b[1mLocal:\x1b[0m   \x1b[36mhttp://localhost:5666/\x1b[0m";
                string dotnetLog = "Now listening on: http://localhost:5023";
                string timeLog = "[11:21:41 INF] Application started.";

                int? vitePort = AppsManager.ViewModels.MainViewModel.ExtractPortFromLog(viteLog);
                int? coloredVitePort = AppsManager.ViewModels.MainViewModel.ExtractPortFromLog(coloredViteLog);
                int? dotnetPort = AppsManager.ViewModels.MainViewModel.ExtractPortFromLog(dotnetLog);
                int? timePort = AppsManager.ViewModels.MainViewModel.ExtractPortFromLog(timeLog);

                Log($"  - Vite 纯文本端口提取 (预期 5666): {vitePort}");
                Log($"  - Vite 彩色文本端口提取 (预期 5666): {coloredVitePort}");
                Log($"  - DotNet 端口提取 (预期 5023): {dotnetPort}");
                Log($"  - 时间戳误判提取 (预期 null): {(timePort.HasValue ? timePort.Value.ToString() : "null")}");

                if (vitePort != 5666 || coloredVitePort != 5666 || dotnetPort != 5023 || timePort.HasValue)
                {
                    throw new Exception("日志端口智能提取纠偏器的正则捕获结果与期望不符。");
                }
                Log("测试 5 通过了！");

                Log(">> 所有自诊断自测试全部成功！核心逻辑验证无误。");
            }
            catch (Exception ex)
            {
                Log($">> 诊断测试中遇到错误: {ex.Message}");
            }
        }
    }
}


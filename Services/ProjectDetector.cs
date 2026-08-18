using System;
using System.Collections.Generic;
using System.IO;
using System.Text.Json;
using System.Text.RegularExpressions;
using AppsManager.Models;

namespace AppsManager.Services
{
    public static class ProjectDetector
    {
        public static ProjectItem DetectProject(string folderPath)
        {
            var item = new ProjectItem
            {
                Id = Guid.NewGuid(),
                Path = folderPath,
                Name = Path.GetFileName(folderPath.TrimEnd(Path.DirectorySeparatorChar))
            };

            if (!Directory.Exists(folderPath)) return item;

            // 搜索所有合法的子服务路径
            var subServices = new List<SubService>();
            ScanDirectory(folderPath, folderPath, 0, subServices);

            foreach (var sub in subServices)
            {
                item.SubServices.Add(sub);
            }

            return item;
        }

        private static void ScanDirectory(string rootPath, string currentPath, int depth, List<SubService> results)
        {
            if (depth > 2) return; // 限制深度为 2 层

            // 1. 检查当前目录是否是一个子服务
            if (TryDetectSingleService(currentPath, out var subService))
            {
                results.Add(subService);
                // 如果当前目录已经被探测为项目，其子目录一般不再递归（防止嵌套识别）
                return;
            }

            // 2. 递归扫描子目录
            try
            {
                foreach (var dir in Directory.GetDirectories(currentPath))
                {
                    string name = Path.GetFileName(dir).ToLower();
                    if (name.StartsWith(".") || name == "node_modules" || name == "bin" || name == "obj" || name == "dist" || name == "packages")
                    {
                        continue;
                    }
                    ScanDirectory(rootPath, dir, depth + 1, results);
                }
            }
            catch { }
        }

        private static bool TryDetectSingleService(string path, out SubService sub)
        {
            sub = new SubService
            {
                Id = Guid.NewGuid(),
                Path = path,
                Name = Path.GetFileName(path)
            };

            // A. Node.js
            string pkgJsonPath = Path.Combine(path, "package.json");
            if (File.Exists(pkgJsonPath))
            {
                DetectNode(path, pkgJsonPath, sub);
                return true;
            }

            // B. C# .NET
            var csprojFiles = Directory.GetFiles(path, "*.csproj", SearchOption.TopDirectoryOnly);
            if (csprojFiles.Length > 0 && IsValidExecutableDotNetProject(csprojFiles[0]))
            {
                DetectDotNet(path, sub);
                return true;
            }

            // C. Python Django
            if (File.Exists(Path.Combine(path, "manage.py")))
            {
                sub.StartCommand = "python manage.py runserver";
                sub.Port = 8000;
                return true;
            }

            // D. Go
            if (File.Exists(Path.Combine(path, "go.mod")))
            {
                sub.StartCommand = "go run .";
                return true;
            }

            return false;
        }

        private static void DetectNode(string folder, string pkgJsonPath, SubService sub)
        {
            try
            {
                string json = File.ReadAllText(pkgJsonPath);
                using var doc = JsonDocument.Parse(json);
                var root = doc.RootElement;
                if (root.TryGetProperty("name", out var nameProp))
                {
                    sub.Name = nameProp.GetString() ?? sub.Name;
                }

                string manager = "npm run";
                if (File.Exists(Path.Combine(folder, "pnpm-lock.yaml")) || File.Exists(Path.Combine(Directory.GetParent(folder)?.FullName ?? "", "pnpm-lock.yaml"))) manager = "pnpm";
                else if (File.Exists(Path.Combine(folder, "yarn.lock")) || File.Exists(Path.Combine(Directory.GetParent(folder)?.FullName ?? "", "yarn.lock"))) manager = "yarn";

                string script = "dev";
                if (root.TryGetProperty("scripts", out var scriptsProp))
                {
                    // 优先匹配包含 dev:xx 的脚本（例如 dev:ip）
                    bool matched = false;
                    foreach (var prop in scriptsProp.EnumerateObject())
                    {
                        if (prop.Name.StartsWith("dev:"))
                        {
                            script = prop.Name;
                            matched = true;
                            break;
                        }
                    }
                    if (!matched)
                    {
                        if (scriptsProp.TryGetProperty("dev", out _)) script = "dev";
                        else if (scriptsProp.TryGetProperty("start", out _)) script = "start";
                    }
                }

                sub.StartCommand = manager == "npm run" ? $"npm run {script}" : $"{manager} {script}";
                sub.Port = 5173;

                // 尝试提取配置端口
                string[] configs = { "vite.config.ts", "vite.config.js", "nuxt.config.ts" };
                foreach (var cfg in configs)
                {
                    string path = Path.Combine(folder, cfg);
                    if (File.Exists(path))
                    {
                        string content = File.ReadAllText(path);
                        var match = Regex.Match(content, @"port\s*:\s*(\d+)");
                        if (match.Success && int.TryParse(match.Groups[1].Value, out int port))
                        {
                            sub.Port = port;
                            return;
                        }
                    }
                }
            }
            catch { }
        }

        private static void DetectDotNet(string folder, SubService sub)
        {
            sub.StartCommand = "dotnet run";
            sub.Port = 5000;

            try
            {
                string launchSettingsPath = Path.Combine(folder, "Properties", "launchSettings.json");
                if (File.Exists(launchSettingsPath))
                {
                    string content = File.ReadAllText(launchSettingsPath);
                    var match = Regex.Match(content, @"localhost:(\d+)");
                    if (match.Success && int.TryParse(match.Groups[1].Value, out int port))
                    {
                        sub.Port = port;
                    }
                }
            }
            catch { }
        }

        private static bool IsValidExecutableDotNetProject(string csprojPath)
        {
            try
            {
                if (!File.Exists(csprojPath)) return false;
                string content = File.ReadAllText(csprojPath);
                if (content.Contains("Sdk=\"Microsoft.NET.Sdk.Web\"") || content.Contains("Sdk=\"Microsoft.NET.Sdk.Worker\""))
                {
                    return true;
                }
                if (content.Contains("<OutputType>Exe</OutputType>") || content.Contains("<OutputType>WinExe</OutputType>"))
                {
                    return true;
                }
            }
            catch { }
            return false;
        }

        public static SubService ParseCommand(string commandLine, string projectRootPath)
        {
            var sub = new SubService
            {
                Id = Guid.NewGuid(),
                Path = projectRootPath,
                Name = "自定义服务",
                StartCommand = commandLine.Trim()
            };

            string cmd = commandLine.Trim();
            if (string.IsNullOrEmpty(cmd)) return sub;

            // 1. 解析 cd 路径
            string workPath = projectRootPath;
            var cdMatch = Regex.Match(cmd, @"cd\s+[""']?([^""'&;]+)[""']?");
            if (cdMatch.Success)
            {
                string relativeOrAbsolute = cdMatch.Groups[1].Value.Trim();
                try
                {
                    if (Path.IsPathRooted(relativeOrAbsolute))
                    {
                        workPath = Path.GetFullPath(relativeOrAbsolute);
                    }
                    else
                    {
                        workPath = Path.GetFullPath(Path.Combine(projectRootPath, relativeOrAbsolute));
                    }
                }
                catch { }
            }
            sub.Path = workPath;
            sub.Name = Path.GetFileName(workPath.TrimEnd(Path.DirectorySeparatorChar)) ?? "自定义服务";

            // 2. 剥离 cd 语句，留下实际命令
            // 例如：cd web && pnpm dev -> pnpm dev
            string remainingCmd = cmd;
            if (cdMatch.Success)
            {
                int index = cmd.IndexOf(cdMatch.Value) + cdMatch.Value.Length;
                remainingCmd = cmd.Substring(index).Trim();
                // 剥离连接符如 && 或 ;
                if (remainingCmd.StartsWith("&&")) remainingCmd = remainingCmd.Substring(2).Trim();
                else if (remainingCmd.StartsWith(";")) remainingCmd = remainingCmd.Substring(1).Trim();
            }
            sub.StartCommand = remainingCmd;

            // 3. 提取端口
            var portMatch = Regex.Match(cmd, @"localhost:(\d+)|--port\s+(\d+)|-p\s+(\d+)");
            if (portMatch.Success)
            {
                for (int i = 1; i <= 3; i++)
                {
                    if (portMatch.Groups[i].Success && int.TryParse(portMatch.Groups[i].Value, out int port))
                    {
                        sub.Port = port;
                        break;
                    }
                }
            }

            return sub;
        }
    }
}

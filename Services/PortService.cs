using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.IO;

namespace AppsManager.Services
{
    public static class PortService
    {
        public static bool IsPortInUse(int port)
        {
            if (port <= 0 || port > 65535) return false;
            return GetPidsByPort(port).Count > 0;
        }

        public static List<int> GetPidsByPort(int port)
        {
            var pids = new List<int>();
            if (port <= 0 || port > 65535) return pids;

            try
            {
                var startInfo = new ProcessStartInfo
                {
                    FileName = "cmd.exe",
                    Arguments = $"/c netstat -ano -p tcp | findstr /R \":{port}\\>\"",
                    RedirectStandardOutput = true,
                    RedirectStandardError = true,
                    UseShellExecute = false,
                    CreateNoWindow = true
                };

                using var process = Process.Start(startInfo);
                if (process != null)
                {
                    string output = process.StandardOutput.ReadToEnd();
                    process.WaitForExit(3000);

                    var lines = output.Split(new[] { "\r\n", "\n" }, StringSplitOptions.RemoveEmptyEntries);
                    foreach (var line in lines)
                    {
                        var parts = line.Split(new[] { ' ' }, StringSplitOptions.RemoveEmptyEntries);
                        if (parts.Length >= 5)
                        {
                            // netstat输出格式: TCP 127.0.0.1:port 0.0.0.0:0 LISTENING PID
                            // 检查地址部分是否以 :port 结尾
                            string localAddress = parts[1];
                            if (localAddress.EndsWith($":{port}"))
                            {
                                string pidStr = parts[parts.Length - 1];
                                if (int.TryParse(pidStr, out int pid) && pid > 0)
                                {
                                    if (!pids.Contains(pid))
                                    {
                                        pids.Add(pid);
                                    }
                                }
                            }
                        }
                    }
                }
            }
            catch (Exception ex)
            {
                Debug.WriteLine($"获取端口 {port} PID 失败: {ex.Message}");
            }

            return pids;
        }

        public static void KillProcessOnPort(int port)
        {
            var pids = GetPidsByPort(port);
            foreach (var pid in pids)
            {
                try
                {
                    // 排除系统进程和本进程
                    if (pid == 0 || pid == 4 || pid == Process.GetCurrentProcess().Id) continue;

                    using var proc = Process.GetProcessById(pid);
                    proc.Kill(true); // true 代表终止进程及其所有子进程（进程树）
                    Debug.WriteLine($"成功杀掉占用端口 {port} 的进程 PID: {pid}");
                }
                catch (Exception ex)
                {
                    Debug.WriteLine($"杀掉占用端口 {port} 的进程 PID: {pid} 失败: {ex.Message}");
                    // 兜底使用 taskkill 命令行强杀
                    try
                    {
                        using var forceKill = Process.Start(new ProcessStartInfo
                        {
                            FileName = "taskkill.exe",
                            Arguments = $"/F /T /PID {pid}",
                            CreateNoWindow = true,
                            UseShellExecute = false
                        });
                        forceKill?.WaitForExit(2000);
                    }
                    catch { }
                }
            }
        }
    }
}

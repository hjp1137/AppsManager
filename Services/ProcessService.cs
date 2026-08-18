using System;
using System.Collections.Concurrent;
using System.Diagnostics;
using System.Text;
using System.Threading.Tasks;
using AppsManager.Models;

namespace AppsManager.Services
{
    public class ProcessService
    {
        private readonly ConcurrentDictionary<Guid, Process> _runningProcesses = new();
        private readonly ConcurrentDictionary<Guid, StringBuilder> _logs = new();

        public event Action<Guid, string>? LogReceived;
        public event Action<Guid, ProjectStatus>? StatusChanged;

        public void StartSubService(SubService item)
        {
            if (item.Status == ProjectStatus.Running || item.Status == ProjectStatus.Starting) return;

            _logs.TryAdd(item.Id, new StringBuilder());
            _logs[item.Id].Clear();
            item.LogContent = string.Empty;

            StatusChanged?.Invoke(item.Id, ProjectStatus.Starting);

            Task.Run(() =>
            {
                try
                {
                    var startInfo = new ProcessStartInfo
                    {
                        FileName = "cmd.exe",
                        Arguments = $"/c {item.StartCommand}",
                        WorkingDirectory = item.Path,
                        RedirectStandardOutput = true,
                        RedirectStandardError = true,
                        UseShellExecute = false,
                        CreateNoWindow = true,
                        StandardOutputEncoding = Encoding.UTF8,
                        StandardErrorEncoding = Encoding.UTF8
                    };

                    var process = new Process { StartInfo = startInfo };
                    process.EnableRaisingEvents = true;

                    process.OutputDataReceived += (s, e) => AppendLog(item.Id, e.Data);
                    process.ErrorDataReceived += (s, e) => AppendLog(item.Id, e.Data);
                    process.Exited += (s, e) => HandleProcessExit(item.Id);

                    if (process.Start())
                    {
                        item.ProcessId = process.Id;
                        _runningProcesses[item.Id] = process;
                        process.BeginOutputReadLine();
                        process.BeginErrorReadLine();
                        if (item.Port == 0)
                        {
                            StatusChanged?.Invoke(item.Id, ProjectStatus.Running);
                        }
                    }
                    else
                    {
                        AppendLog(item.Id, "进程启动失败。");
                        StatusChanged?.Invoke(item.Id, ProjectStatus.Stopped);
                    }
                }
                catch (Exception ex)
                {
                    AppendLog(item.Id, $"启动异常: {ex.Message}");
                    StatusChanged?.Invoke(item.Id, ProjectStatus.Stopped);
                }
            });
        }

        public void StopSubService(SubService item)
        {
            AppendLog(item.Id, "正在停止服务...");

            Task.Run(() =>
            {
                try
                {
                    if (_runningProcesses.TryRemove(item.Id, out var process))
                    {
                        try
                        {
                            if (!process.HasExited) process.Kill(true);
                        }
                        catch { }
                        finally { process.Dispose(); }
                    }

                    if (item.Port > 0)
                    {
                        try
                        {
                            PortService.KillProcessOnPort(item.Port);
                        }
                        catch (Exception ex)
                        {
                            AppendLog(item.Id, $"[AppsManager] 释放端口 {item.Port} 失败: {ex.Message}");
                        }
                    }
                }
                catch (Exception ex)
                {
                    AppendLog(item.Id, $"停止服务异常: {ex.Message}");
                }
                finally
                {
                    StatusChanged?.Invoke(item.Id, ProjectStatus.Stopped);
                    AppendLog(item.Id, "服务已停止。");
                }
            });
        }

        public void StopAll()
        {
            foreach (var id in _runningProcesses.Keys)
            {
                if (_runningProcesses.TryRemove(id, out var process))
                {
                    try
                    {
                        if (!process.HasExited) process.Kill(true);
                    }
                    catch { }
                    finally { process.Dispose(); }
                }
            }
        }

        private void AppendLog(Guid id, string? data)
        {
            if (data == null) return;
            if (!_logs.TryGetValue(id, out var sb))
            {
                sb = new StringBuilder();
                _logs[id] = sb;
            }
            if (sb.Length > 200000) sb.Remove(0, 100000);
            sb.AppendLine(data);
            LogReceived?.Invoke(id, data);
        }

        private void HandleProcessExit(Guid id)
        {
            _runningProcesses.TryRemove(id, out _);
            StatusChanged?.Invoke(id, ProjectStatus.Stopped);
        }
    }
}

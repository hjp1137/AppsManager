using System;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.Diagnostics;
using System.IO;
using System.Linq;
using System.Runtime.CompilerServices;
using System.Windows.Threading;
using AppsManager.Models;
using AppsManager.Services;

namespace AppsManager.ViewModels
{
    public class MainViewModel : INotifyPropertyChanged
    {
        private readonly ProcessService _processService;
        public ProcessService ProcessService => _processService;
        private readonly DispatcherTimer _portTimer;
        private ObservableCollection<ProjectItem> _projects;
        private ObservableCollection<ProjectItem> _filteredProjects;
        private ObservableCollection<string> _groups;
        private string _selectedGroup = "全部";
        private ProjectItem? _selectedProject;

        public ObservableCollection<ProjectItem> Projects
        {
            get => _projects;
            set => SetProperty(ref _projects, value);
        }

        public ObservableCollection<ProjectItem> FilteredProjects
        {
            get => _filteredProjects;
            set => SetProperty(ref _filteredProjects, value);
        }

        public ObservableCollection<string> Groups
        {
            get => _groups;
            set => SetProperty(ref _groups, value);
        }

        public string SelectedGroup
        {
            get => _selectedGroup;
            set
            {
                if (SetProperty(ref _selectedGroup, value))
                {
                    ApplyFilter();
                }
            }
        }

        public ProjectItem? SelectedProject
        {
            get => _selectedProject;
            set
            {
                if (SetProperty(ref _selectedProject, value))
                {
                    foreach (var proj in Projects)
                    {
                        proj.IsSelected = (proj == value);
                    }
                }
            }
        }

        public event PropertyChangedEventHandler? PropertyChanged;

        public MainViewModel()
        {
            _processService = new ProcessService();
            _projects = new ObservableCollection<ProjectItem>();
            _filteredProjects = new ObservableCollection<ProjectItem>();
            _groups = new ObservableCollection<string> { "全部" };

            // 初始化定时器
            _portTimer = new DispatcherTimer
            {
                Interval = TimeSpan.FromSeconds(2)
            };
            _portTimer.Tick += PortTimer_Tick;

            InitializeCommands();
            
            // 绑定日志和状态事件
            _processService.LogReceived += ProcessService_LogReceived;
            _processService.StatusChanged += ProcessService_StatusChanged;

            LoadProjects();
            _portTimer.Start();
        }

        private void PortTimer_Tick(object? sender, EventArgs e)
        {
            foreach (var proj in _projects)
            {
                foreach (var sub in proj.SubServices)
                {
                    if (sub.Port > 0)
                    {
                        bool inUse = PortService.IsPortInUse(sub.Port);
                        if (inUse)
                        {
                            if (sub.Status != ProjectStatus.Running)
                            {
                                App.Current.Dispatcher.Invoke(() => sub.Status = ProjectStatus.Running);
                            }
                        }
                        else
                        {
                            if (sub.Status == ProjectStatus.Running)
                            {
                                App.Current.Dispatcher.Invoke(() => sub.Status = ProjectStatus.Stopped);
                            }
                        }
                    }
                }
            }
        }

        private void ProcessService_LogReceived(Guid subServiceId, string log)
        {
            foreach (var proj in _projects)
            {
                var sub = proj.SubServices.FirstOrDefault(s => s.Id == subServiceId);
                if (sub != null)
                {
                    App.Current.Dispatcher.Invoke(() =>
                    {
                        sub.LogContent += log + "\n";

                        // 智能解析端口纠偏
                        int? parsedPort = ExtractPortFromLog(log);
                        if (parsedPort.HasValue && parsedPort.Value >= 1000 && parsedPort.Value <= 65535)
                        {
                            if (sub.Port != parsedPort.Value)
                            {
                                sub.Port = parsedPort.Value;
                                SaveProjects();
                            }
                        }
                    });
                    break;
                }
            }
        }

        public static int? ExtractPortFromLog(string logLine)
        {
            if (string.IsNullOrEmpty(logLine)) return null;

            // 剥离所有 ANSI 颜色控制字符，防止它们切割破坏端口文本
            string cleanLine = System.Text.RegularExpressions.Regex.Replace(logLine, @"(?:\x1B|\\x1b|\\u001b)\[[0-9;]*m", "");

            var match1 = System.Text.RegularExpressions.Regex.Match(cleanLine, @"listening on\s+(?:https?://)?[^\s:]+:(\d{4,5})", System.Text.RegularExpressions.RegexOptions.IgnoreCase);
            if (match1.Success && int.TryParse(match1.Groups[1].Value, out int p1)) return p1;

            var match2 = System.Text.RegularExpressions.Regex.Match(cleanLine, @"(?:localhost|127\.0\.0\.1|0\.0\.0\.0):(\d{4,5})", System.Text.RegularExpressions.RegexOptions.IgnoreCase);
            if (match2.Success && int.TryParse(match2.Groups[1].Value, out int p2)) return p2;

            var match3 = System.Text.RegularExpressions.Regex.Match(cleanLine, @"local:\s+(?:https?://)?[^\s:]+:(\d{4,5})", System.Text.RegularExpressions.RegexOptions.IgnoreCase);
            if (match3.Success && int.TryParse(match3.Groups[1].Value, out int p3)) return p3;

            var match4 = System.Text.RegularExpressions.Regex.Match(cleanLine, @"\bport\s+(\d{4,5})\b", System.Text.RegularExpressions.RegexOptions.IgnoreCase);
            if (match4.Success && int.TryParse(match4.Groups[1].Value, out int p4)) return p4;

            return null;
        }

        private void ProcessService_StatusChanged(Guid subServiceId, ProjectStatus status)
        {
            foreach (var proj in _projects)
            {
                var sub = proj.SubServices.FirstOrDefault(s => s.Id == subServiceId);
                if (sub != null)
                {
                    App.Current.Dispatcher.Invoke(() => sub.Status = status);
                    break;
                }
            }
        }

        public void LoadProjects()
        {
            var list = ConfigService.LoadProjects();
            Projects.Clear();
            foreach (var item in list)
            {
                Projects.Add(item);
            }
            RefreshGroups();
            ApplyFilter();
        }

        public void SaveProjects()
        {
            ConfigService.SaveProjects(Projects.ToList());
            RefreshGroups();
        }

        private void RefreshGroups()
        {
            var currentGroup = SelectedGroup;
            Groups.Clear();
            Groups.Add("全部");
            foreach (var g in Projects.Select(p => p.Group).Distinct())
            {
                if (!string.IsNullOrEmpty(g)) Groups.Add(g);
            }
            if (Groups.Contains(currentGroup)) SelectedGroup = currentGroup;
            else SelectedGroup = "全部";
        }

        private void ApplyFilter()
        {
            FilteredProjects.Clear();
            var query = SelectedGroup == "全部" 
                ? Projects 
                : Projects.Where(p => p.Group == SelectedGroup);

            foreach (var item in query)
            {
                FilteredProjects.Add(item);
            }
        }

        // 弹窗回调委托，供 View 层注册
        public Func<ProjectItem?, ProjectItem?>? ShowProjectDialog { get; set; }

        private SubService? _selectedSubService;
        public SubService? SelectedSubService
        {
            get => _selectedSubService;
            set => SetProperty(ref _selectedSubService, value);
        }

        public RelayCommand<SubService> StartSubServiceCommand { get; private set; } = null!;
        public RelayCommand<SubService> StopSubServiceCommand { get; private set; } = null!;
        public RelayCommand<SubService> KillPortCommand { get; private set; } = null!;
        public RelayCommand<SubService> RestartSubServiceCommand { get; private set; } = null!;
        public RelayCommand<ProjectItem> StartAllCommand { get; private set; } = null!;
        public RelayCommand<ProjectItem> StopAllCommand { get; private set; } = null!;
        public RelayCommand StopAllGlobalCommand { get; private set; } = null!;
        public RelayCommand<SubService> OpenFolderCommand { get; private set; } = null!;
        public RelayCommand<SubService> OpenVSCodeCommand { get; private set; } = null!;
        public RelayCommand AddProjectCommand { get; private set; } = null!;
        public RelayCommand<ProjectItem> EditProjectCommand { get; private set; } = null!;
        public RelayCommand<ProjectItem> DeleteProjectCommand { get; private set; } = null!;

        private void InitializeCommands()
        {
            StartSubServiceCommand = new RelayCommand<SubService>(s => _processService.StartSubService(s));
            StopSubServiceCommand = new RelayCommand<SubService>(s => _processService.StopSubService(s));
            KillPortCommand = new RelayCommand<SubService>(s => KillPortConflict(s));
            RestartSubServiceCommand = new RelayCommand<SubService>(s => RestartSubService(s));
            StartAllCommand = new RelayCommand<ProjectItem>(p =>
            {
                foreach (var s in p.SubServices) _processService.StartSubService(s);
            });
            StopAllCommand = new RelayCommand<ProjectItem>(p =>
            {
                foreach (var s in p.SubServices) _processService.StopSubService(s);
            });
            StopAllGlobalCommand = new RelayCommand(_ =>
            {
                foreach (var proj in Projects)
                {
                    foreach (var s in proj.SubServices)
                    {
                        _processService.StopSubService(s);
                    }
                }
            });
            OpenFolderCommand = new RelayCommand<SubService>(s =>
            {
                if (Directory.Exists(s.Path))
                {
                    Process.Start(new ProcessStartInfo("explorer.exe", s.Path) { UseShellExecute = true });
                }
            });
            OpenVSCodeCommand = new RelayCommand<SubService>(s =>
            {
                if (Directory.Exists(s.Path))
                {
                    Process.Start(new ProcessStartInfo
                    {
                        FileName = "cmd.exe",
                        Arguments = $"/c code .",
                        WorkingDirectory = s.Path,
                        CreateNoWindow = true,
                        UseShellExecute = false
                    });
                }
            });
            AddProjectCommand = new RelayCommand(_ =>
            {
                var result = ShowProjectDialog?.Invoke(null);
                if (result != null)
                {
                    Projects.Add(result);
                    SaveProjects();
                    ApplyFilter();
                }
            });
            EditProjectCommand = new RelayCommand<ProjectItem>(p =>
            {
                var result = ShowProjectDialog?.Invoke(p);
                if (result != null)
                {
                    p.Name = result.Name;
                    p.Path = result.Path;
                    p.Group = result.Group;
                    p.SubServices.Clear();
                    foreach (var s in result.SubServices) p.SubServices.Add(s);
                    SaveProjects();
                    ApplyFilter();
                }
            });
            DeleteProjectCommand = new RelayCommand<ProjectItem>(p =>
            {
                foreach (var s in p.SubServices)
                {
                    if (s.Status != ProjectStatus.Stopped) _processService.StopSubService(s);
                }
                Projects.Remove(p);
                SaveProjects();
                ApplyFilter();
            });
        }

        private void KillPortConflict(SubService sub)
        {
            if (sub == null || sub.Port <= 0) return;

            App.Current.Dispatcher.Invoke(() =>
            {
                sub.LogContent += $"[AppsManager] 正在强行释放端口 {sub.Port} 的冲突进程...\n";
            });

            System.Threading.Tasks.Task.Run(() =>
            {
                try
                {
                    PortService.KillProcessOnPort(sub.Port);
                    App.Current.Dispatcher.Invoke(() =>
                    {
                        sub.LogContent += $"[AppsManager] 端口 {sub.Port} 上的冲突进程已强行终止释放！\n";
                    });
                }
                catch (Exception ex)
                {
                    App.Current.Dispatcher.Invoke(() =>
                    {
                        sub.LogContent += $"[AppsManager] 强杀端口 {sub.Port} 失败: {ex.Message}\n";
                    });
                }
            });
        }

        private void RestartSubService(SubService sub)
        {
            if (sub == null) return;

            System.Threading.Tasks.Task.Run(async () =>
            {
                _processService.StopSubService(sub);

                // 循环轮询直至子服务状态变回 Stopped，最多等待 5 秒
                int waitCount = 0;
                while (sub.Status != ProjectStatus.Stopped && waitCount < 25)
                {
                    await System.Threading.Tasks.Task.Delay(200);
                    waitCount++;
                }

                // 给端口完全释放多留 300 毫秒缓冲
                await System.Threading.Tasks.Task.Delay(300);

                _processService.StartSubService(sub);
            });
        }

        protected bool SetProperty<T>(ref T storage, T value, [CallerMemberName] string? propertyName = null)
        {
            if (Equals(storage, value)) return false;
            storage = value;
            PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(propertyName));
            return true;
        }
    }
}

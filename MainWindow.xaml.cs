using System;
using System.ComponentModel;
using System.Linq;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Documents;
using System.Windows.Input;
using AppsManager.Models;
using AppsManager.Services;
using AppsManager.ViewModels;

namespace AppsManager
{
    public partial class MainWindow : Window
    {
        private readonly MainViewModel _viewModel;

        public MainWindow()
        {
            InitializeComponent();
            _viewModel = new MainViewModel();
            DataContext = _viewModel;

            // 注册添加/编辑项目的弹窗逻辑
            _viewModel.ShowProjectDialog = item =>
            {
                var dialog = new AddProjectWindow(item) { Owner = this };
                return dialog.ShowDialog() == true ? dialog.ResultProject : null;
            };

            // 订阅 SelectedSubService 变化以切换日志
            _viewModel.PropertyChanged += ViewModel_PropertyChanged;

            // 订阅实时增量日志，进行 ANSI 彩色渲染并追加
            _viewModel.ProcessService.LogReceived += ProcessService_LogReceived;
        }

        private void ViewModel_PropertyChanged(object? sender, PropertyChangedEventArgs e)
        {
            if (e.PropertyName == nameof(MainViewModel.SelectedSubService))
            {
                Dispatcher.Invoke(ReloadLogs);
            }
        }

        private void ProcessService_LogReceived(Guid subServiceId, string logLine)
        {
            Dispatcher.Invoke(() =>
            {
                if (_viewModel.SelectedSubService?.Id == subServiceId)
                {
                    AppendAnsiLine(logLine);
                }
            });
        }

        private void ReloadLogs()
        {
            FlowDoc.Blocks.Clear();
            var selected = _viewModel.SelectedSubService;
            if (selected != null && !string.IsNullOrEmpty(selected.LogContent))
            {
                // 按行拆分现有日志进行首屏绘制
                var lines = selected.LogContent.Split(new[] { "\r\n", "\n" }, StringSplitOptions.None);
                foreach (var line in lines)
                {
                    AppendAnsiLine(line);
                }
            }
        }

        private void AppendAnsiLine(string line)
        {
            // 限制日志行数以保证 UI 性能（控制在最近 1000 行内）
            while (FlowDoc.Blocks.Count > 1000)
            {
                FlowDoc.Blocks.Remove(FlowDoc.Blocks.FirstBlock);
            }

            var paragraph = new Paragraph();
            AnsiColorParser.ParseAndAppend(paragraph, line);
            FlowDoc.Blocks.Add(paragraph);
            TxtLog.ScrollToEnd();
        }

        private void Card_PreviewMouseLeftButtonDown(object sender, MouseButtonEventArgs e)
        {
            if (sender is FrameworkElement element && element.DataContext is ProjectItem item)
            {
                _viewModel.SelectedProject = item;
                if (item.SubServices.Count > 0)
                {
                    _viewModel.SelectedSubService = item.SubServices.FirstOrDefault();
                }
            }
        }

        private void TxtLog_TextChanged(object sender, TextChangedEventArgs e)
        {
            // RichTextBox 没有 TextChanged 事件，我们使用 ScrollToEnd 的方式在增量追加方法里已直接完成
        }

        private void BtnClearLog_Click(object sender, RoutedEventArgs e)
        {
            if (_viewModel.SelectedSubService != null)
            {
                _viewModel.SelectedSubService.LogContent = string.Empty;
                FlowDoc.Blocks.Clear();
            }
        }

        private void Window_Closing(object sender, CancelEventArgs e)
        {
            // 退出应用时强制杀掉所有子服务及其绑定的调试进程
            _viewModel.StopAllGlobalCommand.Execute(null);
        }
    }
}
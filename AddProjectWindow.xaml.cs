using System;
using System.Collections.ObjectModel;
using System.IO;
using System.Windows;
using Microsoft.Win32;
using AppsManager.Models;
using AppsManager.Services;

namespace AppsManager
{
    public partial class AddProjectWindow : Window
    {
        private readonly Guid _existingId;
        public ObservableCollection<SubService> SubServicesList { get; } = new();
        public ProjectItem? ResultProject { get; private set; }

        public AddProjectWindow(ProjectItem? existingProject = null)
        {
            InitializeComponent();
            _existingId = existingProject?.Id ?? Guid.NewGuid();
            DgSubServices.ItemsSource = SubServicesList;

            if (existingProject != null)
            {
                TxtPath.Text = existingProject.Path;
                TxtName.Text = existingProject.Name;
                TxtGroup.Text = existingProject.Group;

                foreach (var s in existingProject.SubServices)
                {
                    SubServicesList.Add(new SubService
                    {
                        Id = s.Id,
                        Name = s.Name,
                        Path = s.Path,
                        StartCommand = s.StartCommand,
                        Port = s.Port,
                        Status = s.Status
                    });
                }
            }
        }

        private void BtnBrowse_Click(object sender, RoutedEventArgs e)
        {
            var dialog = new OpenFolderDialog { Title = "选择项目文件夹" };
            if (dialog.ShowDialog() == true)
            {
                string path = dialog.FolderName;
                TxtPath.Text = path;

                if (string.IsNullOrEmpty(TxtName.Text))
                {
                    TxtName.Text = Path.GetFileName(path.TrimEnd(Path.DirectorySeparatorChar));
                }

                // 智能自动探测所有子服务
                var detected = ProjectDetector.DetectProject(path);
                SubServicesList.Clear();
                foreach (var sub in detected.SubServices)
                {
                    SubServicesList.Add(sub);
                }
            }
        }

        private void BtnParseAdd_Click(object sender, RoutedEventArgs e)
        {
            string cmdLine = TxtCmdPaste.Text.Trim();
            if (string.IsNullOrEmpty(cmdLine))
            {
                MessageBox.Show("请先粘贴或输入启动命令行！", "提示", MessageBoxButton.OK, MessageBoxImage.Warning);
                return;
            }

            string projectRoot = TxtPath.Text.Trim();
            if (string.IsNullOrEmpty(projectRoot) || !Directory.Exists(projectRoot))
            {
                MessageBox.Show("请先确定项目的有效路径！", "提示", MessageBoxButton.OK, MessageBoxImage.Warning);
                return;
            }

            var sub = ProjectDetector.ParseCommand(cmdLine, projectRoot);
            SubServicesList.Add(sub);
            TxtCmdPaste.Text = string.Empty;
        }

        private void BtnAddSub_Click(object sender, RoutedEventArgs e)
        {
            SubServicesList.Add(new SubService
            {
                Id = Guid.NewGuid(),
                Name = "新服务",
                Path = TxtPath.Text.Trim(),
                StartCommand = "npm run dev",
                Port = 0
            });
        }

        private void BtnDeleteSub_Click(object sender, RoutedEventArgs e)
        {
            if (DgSubServices.SelectedItem is SubService selected)
            {
                SubServicesList.Remove(selected);
            }
        }

        private void BtnOk_Click(object sender, RoutedEventArgs e)
        {
            string path = TxtPath.Text.Trim();
            string name = TxtName.Text.Trim();
            string group = TxtGroup.Text.Trim();

            if (string.IsNullOrEmpty(path) || !Directory.Exists(path))
            {
                MessageBox.Show("请输入正确的项目路径！", "提示", MessageBoxButton.OK, MessageBoxImage.Warning);
                return;
            }
            if (string.IsNullOrEmpty(name))
            {
                MessageBox.Show("项目名称不能为空！", "提示", MessageBoxButton.OK, MessageBoxImage.Warning);
                return;
            }

            ResultProject = new ProjectItem
            {
                Id = _existingId,
                Name = name,
                Path = path,
                Group = string.IsNullOrEmpty(group) ? "默认" : group
            };

            foreach (var s in SubServicesList)
            {
                ResultProject.SubServices.Add(s);
            }

            DialogResult = true;
            Close();
        }

        private void BtnCancel_Click(object sender, RoutedEventArgs e)
        {
            DialogResult = false;
            Close();
        }
    }
}

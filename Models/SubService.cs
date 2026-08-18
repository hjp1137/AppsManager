using System;
using System.ComponentModel;
using System.Runtime.CompilerServices;
using System.Text.Json.Serialization;

namespace AppsManager.Models
{
    public enum ProjectStatus
    {
        Stopped,
        Starting,
        Running
    }

    public class SubService : INotifyPropertyChanged
    {
        private Guid _id;
        private string _name = string.Empty;
        private string _path = string.Empty;
        private string _startCommand = string.Empty;
        private int _port;
        private ProjectStatus _status = ProjectStatus.Stopped;
        private string _logContent = string.Empty;

        public Guid Id
        {
            get => _id;
            set => SetProperty(ref _id, value);
        }

        public string Name
        {
            get => _name;
            set => SetProperty(ref _name, value);
        }

        public string Path
        {
            get => _path;
            set => SetProperty(ref _path, value);
        }

        public string StartCommand
        {
            get => _startCommand;
            set => SetProperty(ref _startCommand, value);
        }

        public int Port
        {
            get => _port;
            set => SetProperty(ref _port, value);
        }

        [JsonIgnore]
        public ProjectStatus Status
        {
            get => _status;
            set
            {
                if (SetProperty(ref _status, value))
                {
                    OnPropertyChanged(nameof(StatusBrush));
                    OnPropertyChanged(nameof(StatusText));
                }
            }
        }

        [JsonIgnore]
        public string LogContent
        {
            get => _logContent;
            set
            {
                if (SetProperty(ref _logContent, value))
                {
                    OnPropertyChanged(nameof(StatusBrush));
                }
            }
        }

        [JsonIgnore]
        public System.Windows.Media.Brush StatusBrush => Status switch
        {
            ProjectStatus.Running => System.Windows.Media.Brushes.Green,
            ProjectStatus.Starting => System.Windows.Media.Brushes.Orange,
            _ => string.IsNullOrEmpty(LogContent) 
                ? (System.Windows.Media.Brush)new System.Windows.Media.SolidColorBrush(System.Windows.Media.Color.FromRgb(119, 119, 119)) 
                : System.Windows.Media.Brushes.Red
        };

        [JsonIgnore]
        public string StatusText => Status switch
        {
            ProjectStatus.Running => "运行中",
            ProjectStatus.Starting => "启动中",
            _ => "已停止"
        };

        [JsonIgnore]
        public int? ProcessId { get; set; }

        public event PropertyChangedEventHandler? PropertyChanged;

        protected bool SetProperty<T>(ref T storage, T value, [CallerMemberName] string? propertyName = null)
        {
            if (Equals(storage, value)) return false;
            storage = value;
            OnPropertyChanged(propertyName);
            return true;
        }

        protected void OnPropertyChanged([CallerMemberName] string? propertyName = null)
        {
            PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(propertyName));
        }
    }
}

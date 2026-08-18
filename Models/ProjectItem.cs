using System;
using System.Collections.ObjectModel;
using System.Collections.Specialized;
using System.ComponentModel;
using System.Linq;
using System.Runtime.CompilerServices;
using System.Text.Json.Serialization;

namespace AppsManager.Models
{
    public class ProjectItem : INotifyPropertyChanged
    {
        private Guid _id;
        private string _name = string.Empty;
        private string _path = string.Empty;
        private string _group = "默认";
        private bool _isSelected;
        private ObservableCollection<SubService> _subServices = new();

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

        public string Group
        {
            get => _group;
            set => SetProperty(ref _group, value);
        }

        [JsonIgnore]
        public bool IsSelected
        {
            get => _isSelected;
            set => SetProperty(ref _isSelected, value);
        }

        public ObservableCollection<SubService> SubServices
        {
            get => _subServices;
            set
            {
                if (SetProperty(ref _subServices, value))
                {
                    SubscribeToSubServices();
                }
            }
        }

        [JsonIgnore]
        public string StatusColor
        {
            get
            {
                if (SubServices.Count == 0) return "#F44336";
                if (SubServices.All(s => s.Status == ProjectStatus.Stopped)) return "#F44336"; // 🔴 全部停止
                if (SubServices.All(s => s.Status == ProjectStatus.Running)) return "#4CAF50"; // 🟢 全部运行
                return "#FFC107"; // 🟡 部分启动/运行中
            }
        }

        [JsonIgnore]
        public string StatusText
        {
            get
            {
                if (SubServices.Count == 0) return "无启动项";
                int running = SubServices.Count(s => s.Status == ProjectStatus.Running);
                int total = SubServices.Count;
                return $"运行中 ({running}/{total})";
            }
        }

        public event PropertyChangedEventHandler? PropertyChanged;

        public ProjectItem()
        {
            _subServices.CollectionChanged += SubServices_CollectionChanged;
        }

        private void SubscribeToSubServices()
        {
            foreach (var item in _subServices)
            {
                item.PropertyChanged -= SubService_PropertyChanged;
                item.PropertyChanged += SubService_PropertyChanged;
            }
        }

        private void SubServices_CollectionChanged(object? sender, NotifyCollectionChangedEventArgs e)
        {
            SubscribeToSubServices();
            OnStatusChanged();
        }

        private void SubService_PropertyChanged(object? sender, PropertyChangedEventArgs e)
        {
            if (e.PropertyName == nameof(SubService.Status))
            {
                OnStatusChanged();
            }
        }

        private void OnStatusChanged()
        {
            OnPropertyChanged(nameof(StatusColor));
            OnPropertyChanged(nameof(StatusText));
        }

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

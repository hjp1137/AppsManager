using System;
using System.Collections.Generic;
using System.IO;
using System.Text.Json;
using AppsManager.Models;

namespace AppsManager.Services
{
    public static class ConfigService
    {
        private static readonly string FolderPath = Path.Combine(
            Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData),
            "AppsManager"
        );
        private static readonly string FilePath = Path.Combine(FolderPath, "projects.json");

        public static List<ProjectItem> LoadProjects()
        {
            try
            {
                if (!File.Exists(FilePath))
                {
                    return new List<ProjectItem>();
                }
                string json = File.ReadAllText(FilePath);
                var list = JsonSerializer.Deserialize<List<ProjectItem>>(json);
                return list ?? new List<ProjectItem>();
            }
            catch
            {
                return new List<ProjectItem>();
            }
        }

        public static void SaveProjects(List<ProjectItem> projects)
        {
            try
            {
                if (!Directory.Exists(FolderPath))
                {
                    Directory.CreateDirectory(FolderPath);
                }
                var options = new JsonSerializerOptions { WriteIndented = true };
                string json = JsonSerializer.Serialize(projects, options);
                File.WriteAllText(FilePath, json);
            }
            catch (Exception ex)
            {
                System.Diagnostics.Debug.WriteLine($"保存配置失败: {ex.Message}");
            }
        }
    }
}

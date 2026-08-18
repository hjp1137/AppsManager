using System;
using System.Text.RegularExpressions;
using System.Windows;
using System.Windows.Documents;
using System.Windows.Media;

namespace AppsManager.Services
{
    public static class AnsiColorParser
    {
        public static void ParseAndAppend(Paragraph paragraph, string text)
        {
            if (string.IsNullOrEmpty(text)) return;

            // 剥离/替换可能存在的裸 \r\n 并按行分裂，但由于后台重定向通常是一行行触发，我们在这里只做行内 ANSI 解析
            // 默认控制台画刷配色
            Brush currentForeground = Brushes.LightGray;
            FontWeight currentFontWeight = FontWeights.Normal;

            // 匹配 \x1b[...m 或者 \u001b[...m
            string pattern = @"(?:\x1B|\\x1b|\\u001b)\[([0-9;]*)m";
            var matches = Regex.Matches(text, pattern);
            var parts = Regex.Split(text, pattern);

            for (int i = 0; i < parts.Length; i++)
            {
                if (i % 2 == 0)
                {
                    // 纯文本段
                    string content = parts[i];
                    if (!string.IsNullOrEmpty(content))
                    {
                        var run = new Run(content)
                        {
                            Foreground = currentForeground,
                            FontWeight = currentFontWeight
                        };
                        paragraph.Inlines.Add(run);
                    }
                }
                else
                {
                    // ANSI 状态控制码
                    string codes = parts[i];
                    foreach (var code in codes.Split(';'))
                    {
                        if (int.TryParse(code, out int val))
                        {
                            switch (val)
                            {
                                case 0:
                                    currentForeground = Brushes.LightGray;
                                    currentFontWeight = FontWeights.Normal;
                                    break;
                                case 1:
                                    currentFontWeight = FontWeights.Bold;
                                    break;
                                case 2:
                                    currentForeground = Brushes.Gray;
                                    break;
                                case 22:
                                    currentFontWeight = FontWeights.Normal;
                                    currentForeground = Brushes.LightGray;
                                    break;
                                case 30: currentForeground = Brushes.Black; break;
                                case 31: currentForeground = Brushes.Red; break;
                                case 32: currentForeground = Brushes.Green; break;
                                case 33: currentForeground = Brushes.Yellow; break;
                                case 34: currentForeground = Brushes.Blue; break;
                                case 35: currentForeground = Brushes.Magenta; break;
                                case 36: currentForeground = Brushes.Cyan; break;
                                case 37: currentForeground = Brushes.White; break;
                                case 39:
                                    currentForeground = Brushes.LightGray;
                                    break;
                            }
                        }
                    }
                }
            }
        }
    }
}

using KeepScreenAwake.Core.Config;

namespace KeepScreenAwake.Core.Transport;

public enum CommandType { Status, On, Off, Mode, Logs }

public class Request
{
    public CommandType Command { get; set; }
    public string? Mode { get; set; } // for Mode command
    public int Lines { get; set; } = 50;
}

public class Response
{
    public bool Ok { get; set; }
    public object? Data { get; set; }
    public string? Error { get; set; }
}

public class StatusData
{
    public string Mode { get; set; } = "";
    public bool AwakeActive { get; set; }
    public bool DisplayOnly { get; set; }
    public List<ScheduleWindowData> Schedule { get; set; } = new();
}

public class ScheduleWindowData
{
    public string Start { get; set; } = "";
    public string End { get; set; } = "";
    public List<string> Days { get; set; } = new();
}

public class LogsData
{
    public List<string> Lines { get; set; } = new();
}

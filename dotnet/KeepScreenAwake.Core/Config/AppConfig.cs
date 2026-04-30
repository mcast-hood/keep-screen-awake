using System.Collections.Generic;

namespace KeepScreenAwake.Core.Config;

public class AppConfig
{
    public string Mode { get; set; } = "always"; // always | toggle | schedule
    public List<ScheduleWindow> Schedule { get; set; } = new();
    public bool DisplayOnly { get; set; } = false;
    public IpcConfig Ipc { get; set; } = new();
    public LogConfig Log { get; set; } = new();
}

public class ScheduleWindow
{
    public string Start { get; set; } = "09:00"; // HH:mm
    public string End { get; set; } = "18:00";
    public List<string> Days { get; set; } = new() { "Mon", "Tue", "Wed", "Thu", "Fri" };
}

public class IpcConfig
{
    public string PipeName { get; set; } = "keep-screen-awake";
    public int HttpPort { get; set; } = 9877;
}

public class LogConfig
{
    public string Level { get; set; } = "Information";
    public string File { get; set; } = "";
}

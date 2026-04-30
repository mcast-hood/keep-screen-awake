using System.Collections.Concurrent;
using System.Text.Json;
using KeepScreenAwake.Core.Awake;
using KeepScreenAwake.Core.Config;
using KeepScreenAwake.Core.Transport;

namespace KeepScreenAwake.Service;

public sealed class Worker : BackgroundService
{
    private readonly ILogger<Worker> _logger;

    private AppConfig _config = new();
    private string _mode = "always";
    private bool _displayOnly;
    private List<ScheduleWindow> _schedule = new();

    private AwakeManager? _awakeManager;
    private ITransportServer? _transportServer;

    private readonly ConcurrentQueue<string> _logBuffer = new();
    private const int MaxLogLines = 1000;

    public Worker(ILogger<Worker> logger)
    {
        _logger = logger;
    }

    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        var configPath = Path.Combine(
            AppContext.BaseDirectory, "config.yaml");

        _config = ConfigLoader.LoadOrDefault(configPath);
        _mode = _config.Mode;
        _displayOnly = _config.DisplayOnly;
        _schedule = _config.Schedule;

        Log($"Starting KeepScreenAwake. Mode={_mode}, DisplayOnly={_displayOnly}");

        _awakeManager = new AwakeManager(_displayOnly);
        _transportServer = TransportFactory.CreateServer(_config);

        // Apply initial awake state
        if (_mode is "always" or "toggle")
        {
            _awakeManager.Enable();
            Log("Awake enabled.");
        }

        // Start IPC server
        var ipcTask = _transportServer.ServeAsync(HandleRequestAsync, stoppingToken);

        // Schedule timer if needed
        if (_mode == "schedule")
        {
            var scheduleTask = RunScheduleLoopAsync(stoppingToken);
            await Task.WhenAll(ipcTask, scheduleTask).ConfigureAwait(false);
        }
        else
        {
            await ipcTask.ConfigureAwait(false);
        }

        Log("Worker stopping. Disabling awake...");
        _awakeManager.Disable();
    }

    private async Task RunScheduleLoopAsync(CancellationToken ct)
    {
        using var timer = new PeriodicTimer(TimeSpan.FromSeconds(30));
        ApplySchedule();

        while (await timer.WaitForNextTickAsync(ct).ConfigureAwait(false))
        {
            ApplySchedule();
        }
    }

    private void ApplySchedule()
    {
        if (_awakeManager is null) return;

        var shouldBeActive = IsInScheduleWindow(_schedule, DateTime.Now);
        if (shouldBeActive && !_awakeManager.IsActive)
        {
            _awakeManager.Enable();
            Log("Schedule: entered active window, awake enabled.");
        }
        else if (!shouldBeActive && _awakeManager.IsActive)
        {
            _awakeManager.Disable();
            Log("Schedule: exited active window, awake disabled.");
        }
    }

    private Task<Response> HandleRequestAsync(Request request)
    {
        try
        {
            return Task.FromResult(request.Command switch
            {
                CommandType.Status => HandleStatus(),
                CommandType.On => HandleOn(),
                CommandType.Off => HandleOff(),
                CommandType.Mode => HandleMode(request.Mode),
                CommandType.Logs => HandleLogs(request.Lines),
                _ => new Response { Ok = false, Error = $"Unknown command: {request.Command}" }
            });
        }
        catch (Exception ex)
        {
            return Task.FromResult(new Response { Ok = false, Error = ex.Message });
        }
    }

    private Response HandleStatus()
    {
        var scheduleData = _schedule.Select(s => new ScheduleWindowData
        {
            Start = s.Start,
            End = s.End,
            Days = s.Days
        }).ToList();

        return new Response
        {
            Ok = true,
            Data = new StatusData
            {
                Mode = _mode,
                AwakeActive = _awakeManager?.IsActive ?? false,
                DisplayOnly = _displayOnly,
                Schedule = scheduleData
            }
        };
    }

    private Response HandleOn()
    {
        if (_awakeManager is null)
            return new Response { Ok = false, Error = "Awake manager not initialized" };

        _awakeManager.Enable();
        Log("IPC: awake enabled.");
        return new Response { Ok = true, Data = "Awake enabled" };
    }

    private Response HandleOff()
    {
        if (_awakeManager is null)
            return new Response { Ok = false, Error = "Awake manager not initialized" };

        _awakeManager.Disable();
        Log("IPC: awake disabled.");
        return new Response { Ok = true, Data = "Awake disabled" };
    }

    private Response HandleMode(string? newMode)
    {
        if (string.IsNullOrWhiteSpace(newMode))
            return new Response { Ok = false, Error = "Mode value is required" };

        var lower = newMode.ToLowerInvariant();
        if (lower is not ("always" or "toggle" or "schedule"))
            return new Response { Ok = false, Error = $"Invalid mode '{newMode}'" };

        _mode = lower;
        Log($"IPC: mode changed to '{_mode}'.");

        if (_mode == "always" && _awakeManager is not null && !_awakeManager.IsActive)
            _awakeManager.Enable();

        return new Response { Ok = true, Data = $"Mode set to '{_mode}'" };
    }

    private Response HandleLogs(int lines)
    {
        var all = _logBuffer.ToArray();
        var subset = lines > 0 && lines < all.Length
            ? all[^lines..]
            : all;

        return new Response
        {
            Ok = true,
            Data = new LogsData { Lines = subset.ToList() }
        };
    }

    private void Log(string message)
    {
        var entry = $"[{DateTime.Now:yyyy-MM-dd HH:mm:ss}] {message}";
        _logger.LogInformation("{Message}", message);

        _logBuffer.Enqueue(entry);
        while (_logBuffer.Count > MaxLogLines)
            _logBuffer.TryDequeue(out _);
    }

    /// <summary>
    /// Determines if the given time falls within any configured schedule window.
    /// Public for testing.
    /// </summary>
    public static bool IsInScheduleWindow(IEnumerable<ScheduleWindow> schedule, DateTime now)
    {
        var currentTime = TimeOnly.FromDateTime(now);
        var currentDay = now.DayOfWeek;

        foreach (var window in schedule)
        {
            if (!TryParseDay(currentDay, out var abbrev))
                continue;

            if (!window.Days.Contains(abbrev, StringComparer.OrdinalIgnoreCase))
                continue;

            if (!TimeOnly.TryParse(window.Start, out var start) ||
                !TimeOnly.TryParse(window.End, out var end))
                continue;

            if (start <= end)
            {
                if (currentTime >= start && currentTime < end)
                    return true;
            }
            else
            {
                // Overnight window e.g. 22:00 - 06:00
                if (currentTime >= start || currentTime < end)
                    return true;
            }
        }

        return false;
    }

    private static bool TryParseDay(DayOfWeek day, out string abbrev)
    {
        abbrev = day switch
        {
            DayOfWeek.Monday => "Mon",
            DayOfWeek.Tuesday => "Tue",
            DayOfWeek.Wednesday => "Wed",
            DayOfWeek.Thursday => "Thu",
            DayOfWeek.Friday => "Fri",
            DayOfWeek.Saturday => "Sat",
            DayOfWeek.Sunday => "Sun",
            _ => ""
        };
        return abbrev != "";
    }

    public override void Dispose()
    {
        _awakeManager?.Dispose();
        if (_transportServer is not null)
            _transportServer.DisposeAsync().AsTask().GetAwaiter().GetResult();
        base.Dispose();
    }
}

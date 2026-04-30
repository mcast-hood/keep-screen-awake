using KeepScreenAwake.Core.Config;
using KeepScreenAwake.Service;
using Xunit;

namespace KeepScreenAwake.Tests;

public class ScheduleWindowTests
{
    private static ScheduleWindow MakeWindow(string start, string end, params string[] days) =>
        new ScheduleWindow { Start = start, End = end, Days = days.ToList() };

    [Fact]
    public void InWindow_ReturnsTrue_WhenTimeAndDayMatch()
    {
        // Monday 10:00
        var now = new DateTime(2026, 4, 27, 10, 0, 0); // Monday
        var schedule = new[] { MakeWindow("09:00", "18:00", "Mon", "Tue", "Wed", "Thu", "Fri") };

        var result = Worker.IsInScheduleWindow(schedule, now);

        Assert.True(result);
    }

    [Fact]
    public void OutOfWindow_ReturnsFalse_WhenBeforeStartTime()
    {
        // Monday 08:00 — before start
        var now = new DateTime(2026, 4, 27, 8, 0, 0); // Monday
        var schedule = new[] { MakeWindow("09:00", "18:00", "Mon", "Tue", "Wed", "Thu", "Fri") };

        var result = Worker.IsInScheduleWindow(schedule, now);

        Assert.False(result);
    }

    [Fact]
    public void OutOfWindow_ReturnsFalse_WhenAfterEndTime()
    {
        // Monday 19:00 — after end
        var now = new DateTime(2026, 4, 27, 19, 0, 0); // Monday
        var schedule = new[] { MakeWindow("09:00", "18:00", "Mon", "Tue", "Wed", "Thu", "Fri") };

        var result = Worker.IsInScheduleWindow(schedule, now);

        Assert.False(result);
    }

    [Fact]
    public void OutOfWindow_ReturnsFalse_WhenWrongDay()
    {
        // Saturday 10:00 — weekday-only window
        var now = new DateTime(2026, 4, 25, 10, 0, 0); // Saturday
        var schedule = new[] { MakeWindow("09:00", "18:00", "Mon", "Tue", "Wed", "Thu", "Fri") };

        var result = Worker.IsInScheduleWindow(schedule, now);

        Assert.False(result);
    }

    [Fact]
    public void InWindow_ReturnsTrue_ForWeekendWindow()
    {
        // Saturday 12:00
        var now = new DateTime(2026, 4, 25, 12, 0, 0); // Saturday
        var schedule = new[] { MakeWindow("10:00", "20:00", "Sat", "Sun") };

        var result = Worker.IsInScheduleWindow(schedule, now);

        Assert.True(result);
    }

    [Fact]
    public void EmptySchedule_ReturnsFalse()
    {
        var now = new DateTime(2026, 4, 27, 10, 0, 0);
        var result = Worker.IsInScheduleWindow(Array.Empty<ScheduleWindow>(), now);
        Assert.False(result);
    }

    [Fact]
    public void MultipleWindows_ReturnsTrueIfAnyMatches()
    {
        var now = new DateTime(2026, 4, 27, 14, 0, 0); // Monday 14:00
        var schedule = new[]
        {
            MakeWindow("08:00", "10:00", "Mon"),   // doesn't match (past)
            MakeWindow("12:00", "16:00", "Mon"),   // matches
        };

        var result = Worker.IsInScheduleWindow(schedule, now);

        Assert.True(result);
    }

    [Fact]
    public void OvernightWindow_ReturnsTrue_AfterMidnight()
    {
        // Tuesday 01:00 — in overnight window that started Monday 22:00
        var now = new DateTime(2026, 4, 28, 1, 0, 0); // Tuesday
        var schedule = new[] { MakeWindow("22:00", "06:00", "Tue") };

        var result = Worker.IsInScheduleWindow(schedule, now);

        Assert.True(result);
    }

    [Fact]
    public void ExactStartTime_IsIncluded()
    {
        var now = new DateTime(2026, 4, 27, 9, 0, 0); // Exactly 09:00 Monday
        var schedule = new[] { MakeWindow("09:00", "18:00", "Mon") };

        var result = Worker.IsInScheduleWindow(schedule, now);

        Assert.True(result);
    }

    [Fact]
    public void ExactEndTime_IsExcluded()
    {
        var now = new DateTime(2026, 4, 27, 18, 0, 0); // Exactly 18:00 Monday
        var schedule = new[] { MakeWindow("09:00", "18:00", "Mon") };

        var result = Worker.IsInScheduleWindow(schedule, now);

        Assert.False(result);
    }
}

public class ConfigLoaderTests
{
    [Fact]
    public void LoadOrDefault_ReturnDefaultConfig_WhenFileDoesNotExist()
    {
        var path = Path.Combine(Path.GetTempPath(), $"nonexistent_{Guid.NewGuid()}.yaml");

        var config = ConfigLoader.LoadOrDefault(path);

        Assert.NotNull(config);
        Assert.Equal("always", config.Mode);
        Assert.False(config.DisplayOnly);
        Assert.Empty(config.Schedule);
        Assert.Equal("keep-screen-awake", config.Ipc.PipeName);
        Assert.Equal(9877, config.Ipc.HttpPort);
        Assert.Equal("Information", config.Log.Level);
    }

    [Fact]
    public void LoadOrDefault_ParsesValidYaml()
    {
        var yaml = @"
mode: schedule
displayOnly: true
ipc:
  pipeName: test-pipe
  httpPort: 1234
log:
  level: Debug
schedule:
  - start: ""09:00""
    end: ""17:00""
    days:
      - Mon
      - Fri
";
        var path = Path.Combine(Path.GetTempPath(), $"ksa_test_{Guid.NewGuid()}.yaml");
        File.WriteAllText(path, yaml);

        try
        {
            var config = ConfigLoader.LoadOrDefault(path);

            Assert.Equal("schedule", config.Mode);
            Assert.True(config.DisplayOnly);
            Assert.Equal("test-pipe", config.Ipc.PipeName);
            Assert.Equal(1234, config.Ipc.HttpPort);
            Assert.Equal("Debug", config.Log.Level);
            Assert.Single(config.Schedule);
            Assert.Equal("09:00", config.Schedule[0].Start);
            Assert.Equal("17:00", config.Schedule[0].End);
            Assert.Contains("Mon", config.Schedule[0].Days);
            Assert.Contains("Fri", config.Schedule[0].Days);
        }
        finally
        {
            File.Delete(path);
        }
    }

    [Fact]
    public void LoadOrDefault_ThrowsForInvalidMode()
    {
        var yaml = "mode: invalid-mode\n";
        var path = Path.Combine(Path.GetTempPath(), $"ksa_invalid_{Guid.NewGuid()}.yaml");
        File.WriteAllText(path, yaml);

        try
        {
            Assert.Throws<InvalidOperationException>(() => ConfigLoader.LoadOrDefault(path));
        }
        finally
        {
            File.Delete(path);
        }
    }
}

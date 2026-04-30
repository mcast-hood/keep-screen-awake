using System.CommandLine;
using System.CommandLine.Parsing;
using System.Text.Json;
using KeepScreenAwake.Core.Config;
using KeepScreenAwake.Core.Transport;

var configPath = Path.Combine(AppContext.BaseDirectory, "config.yaml");
var config = ConfigLoader.LoadOrDefault(configPath);

var rootCommand = new RootCommand("ksa — keep-screen-awake CLI");

// ── status ───────────────────────────────────────────────────────────────────
var statusCmd = new Command("status", "Show current awake status");
statusCmd.SetAction(async (ParseResult _, CancellationToken ct) =>
{
    using var client = TransportFactory.CreateClient(config);
    var resp = await SendOrExit(client, new Request { Command = CommandType.Status });
    if (resp is null) return 1;

    var data = DeserializeData<StatusData>(resp);
    if (data is null) return 1;

    Console.WriteLine($"Mode:        {data.Mode}");
    Console.WriteLine($"AwakeActive: {data.AwakeActive}");
    Console.WriteLine($"DisplayOnly: {data.DisplayOnly}");
    if (data.Schedule.Count > 0)
    {
        Console.WriteLine("Schedule:");
        foreach (var w in data.Schedule)
            Console.WriteLine($"  {w.Start}-{w.End} [{string.Join(",", w.Days)}]");
    }
    return 0;
});
rootCommand.Subcommands.Add(statusCmd);

// ── on ───────────────────────────────────────────────────────────────────────
var onCmd = new Command("on", "Enable awake (toggle mode)");
onCmd.SetAction(async (ParseResult _, CancellationToken ct) =>
{
    using var client = TransportFactory.CreateClient(config);
    var resp = await SendOrExit(client, new Request { Command = CommandType.On });
    if (resp is null) return 1;
    Console.WriteLine(resp.Data?.ToString() ?? "OK");
    return 0;
});
rootCommand.Subcommands.Add(onCmd);

// ── off ──────────────────────────────────────────────────────────────────────
var offCmd = new Command("off", "Disable awake");
offCmd.SetAction(async (ParseResult _, CancellationToken ct) =>
{
    using var client = TransportFactory.CreateClient(config);
    var resp = await SendOrExit(client, new Request { Command = CommandType.Off });
    if (resp is null) return 1;
    Console.WriteLine(resp.Data?.ToString() ?? "OK");
    return 0;
});
rootCommand.Subcommands.Add(offCmd);

// ── mode ─────────────────────────────────────────────────────────────────────
var modeArg = new Argument<string>("value");
modeArg.Description = "Mode: always | toggle | schedule";
var modeCmd = new Command("mode", "Switch operating mode");
modeCmd.Add(modeArg);
modeCmd.SetAction(async (ParseResult result, CancellationToken ct) =>
{
    var value = result.GetValue(modeArg)!;
    using var client = TransportFactory.CreateClient(config);
    var resp = await SendOrExit(client, new Request { Command = CommandType.Mode, Mode = value });
    if (resp is null) return 1;
    Console.WriteLine(resp.Data?.ToString() ?? "OK");
    return 0;
});
rootCommand.Subcommands.Add(modeCmd);

// ── logs ─────────────────────────────────────────────────────────────────────
var linesOpt = new Option<int>("--lines");
linesOpt.Description = "Number of log lines to show";
linesOpt.DefaultValueFactory = _ => 50;
var logsCmd = new Command("logs", "Show recent log lines");
logsCmd.Add(linesOpt);
logsCmd.SetAction(async (ParseResult result, CancellationToken ct) =>
{
    var lines = result.GetValue(linesOpt);
    using var client = TransportFactory.CreateClient(config);
    var resp = await SendOrExit(client, new Request { Command = CommandType.Logs, Lines = lines });
    if (resp is null) return 1;

    var data = DeserializeData<LogsData>(resp);
    if (data is null) return 1;

    foreach (var line in data.Lines)
        Console.WriteLine(line);
    return 0;
});
rootCommand.Subcommands.Add(logsCmd);

// ── invoke ───────────────────────────────────────────────────────────────────
var cfg = new CommandLineConfiguration(rootCommand);
return await cfg.InvokeAsync(args);

// ────────────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────────────

static async Task<Response?> SendOrExit(ITransportClient client, Request request)
{
    try
    {
        var response = await client.SendAsync(request);
        if (!response.Ok)
        {
            Console.Error.WriteLine($"Error: {response.Error}");
            return null;
        }
        return response;
    }
    catch (Exception ex)
    {
        Console.Error.WriteLine($"Failed to connect to service: {ex.Message}");
        return null;
    }
}

static T? DeserializeData<T>(Response response) where T : class
{
    if (response.Data is null) return null;

    try
    {
        if (response.Data is JsonElement element)
            return JsonSerializer.Deserialize<T>(element.GetRawText(), JsonOptions.Default);

        if (response.Data is T typed)
            return typed;

        var json = JsonSerializer.Serialize(response.Data, JsonOptions.Default);
        return JsonSerializer.Deserialize<T>(json, JsonOptions.Default);
    }
    catch (Exception ex)
    {
        Console.Error.WriteLine($"Failed to parse response data: {ex.Message}");
        return null;
    }
}

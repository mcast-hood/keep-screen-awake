using KeepScreenAwake.Service;

var builder = Host.CreateDefaultBuilder(args);

if (OperatingSystem.IsWindows())
    builder.UseWindowsService(o => o.ServiceName = "KeepScreenAwake");

builder.ConfigureServices(services => services.AddHostedService<Worker>());

await builder.Build().RunAsync();

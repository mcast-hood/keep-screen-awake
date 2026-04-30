using KeepScreenAwake.Core.Config;

namespace KeepScreenAwake.Core.Transport;

public static class TransportFactory
{
    public static ITransportServer CreateServer(AppConfig config)
    {
        if (OperatingSystem.IsWindows())
            return new NamedPipeServer(config.Ipc.PipeName);

        if (OperatingSystem.IsMacOS())
            return new HttpServer(config.Ipc.HttpPort);

        throw new PlatformNotSupportedException("Transport is only supported on Windows and macOS.");
    }

    public static ITransportClient CreateClient(AppConfig config)
    {
        if (OperatingSystem.IsWindows())
            return new NamedPipeClient(config.Ipc.PipeName);

        if (OperatingSystem.IsMacOS())
            return new HttpTransportClient(config.Ipc.HttpPort);

        throw new PlatformNotSupportedException("Transport is only supported on Windows and macOS.");
    }
}

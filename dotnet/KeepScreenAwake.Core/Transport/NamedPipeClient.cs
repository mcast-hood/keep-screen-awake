using System.IO.Pipes;
using System.Text.Json;

namespace KeepScreenAwake.Core.Transport;

public sealed class NamedPipeClient : ITransportClient
{
    private readonly string _pipeName;
    private readonly int _connectTimeoutMs;
    private bool _disposed;

    public NamedPipeClient(string pipeName, int connectTimeoutMs = 5000)
    {
        if (!OperatingSystem.IsWindows())
            throw new PlatformNotSupportedException("NamedPipeClient is supported on Windows only.");
        _pipeName = pipeName;
        _connectTimeoutMs = connectTimeoutMs;
    }

    public async Task<Response> SendAsync(Request request)
    {
        ObjectDisposedException.ThrowIf(_disposed, this);

        using var pipe = new NamedPipeClientStream(
            ".",
            _pipeName,
            PipeDirection.InOut,
            PipeOptions.Asynchronous);

        await pipe.ConnectAsync(_connectTimeoutMs).ConfigureAwait(false);

        using var reader = new StreamReader(pipe, leaveOpen: true);
        await using var writer = new StreamWriter(pipe, leaveOpen: true) { AutoFlush = true };

        var json = JsonSerializer.Serialize(request, JsonOptions.Default);
        await writer.WriteLineAsync(json).ConfigureAwait(false);

        var responseLine = await reader.ReadLineAsync().ConfigureAwait(false);
        if (responseLine is null)
            return new Response { Ok = false, Error = "No response from server" };

        return JsonSerializer.Deserialize<Response>(responseLine, JsonOptions.Default)
               ?? new Response { Ok = false, Error = "Invalid response" };
    }

    public void Dispose()
    {
        _disposed = true;
    }
}

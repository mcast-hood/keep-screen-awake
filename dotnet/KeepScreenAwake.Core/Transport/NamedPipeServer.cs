using System.IO.Pipes;
using System.Text.Json;

namespace KeepScreenAwake.Core.Transport;

public sealed class NamedPipeServer : ITransportServer
{
    private readonly string _pipeName;
    private const int MaxConnections = 10;

    public NamedPipeServer(string pipeName)
    {
        if (!OperatingSystem.IsWindows())
            throw new PlatformNotSupportedException("NamedPipeServer is supported on Windows only.");
        _pipeName = pipeName;
    }

    public async Task ServeAsync(Func<Request, Task<Response>> handler, CancellationToken ct)
    {
        var tasks = new List<Task>();

        while (!ct.IsCancellationRequested)
        {
            var pipe = new NamedPipeServerStream(
                _pipeName,
                PipeDirection.InOut,
                MaxConnections,
                PipeTransmissionMode.Byte,
                PipeOptions.Asynchronous);

            try
            {
                await pipe.WaitForConnectionAsync(ct).ConfigureAwait(false);
            }
            catch (OperationCanceledException)
            {
                await pipe.DisposeAsync().ConfigureAwait(false);
                break;
            }
            catch
            {
                await pipe.DisposeAsync().ConfigureAwait(false);
                continue;
            }

            var task = HandleConnectionAsync(pipe, handler, ct);
            tasks.Add(task);

            // Clean up completed tasks
            tasks.RemoveAll(t => t.IsCompleted);
        }

        // Wait for all in-flight connections to finish
        if (tasks.Count > 0)
            await Task.WhenAll(tasks).ConfigureAwait(false);
    }

    private static async Task HandleConnectionAsync(
        NamedPipeServerStream pipe,
        Func<Request, Task<Response>> handler,
        CancellationToken ct)
    {
        await using (pipe)
        {
            try
            {
                using var reader = new StreamReader(pipe, leaveOpen: true);
                await using var writer = new StreamWriter(pipe, leaveOpen: true) { AutoFlush = true };

                var line = await reader.ReadLineAsync(ct).ConfigureAwait(false);
                if (line is null) return;

                Request? request = null;
                try
                {
                    request = JsonSerializer.Deserialize<Request>(line, JsonOptions.Default);
                }
                catch
                {
                    var errResponse = new Response { Ok = false, Error = "Invalid JSON request" };
                    await writer.WriteLineAsync(JsonSerializer.Serialize(errResponse, JsonOptions.Default)).ConfigureAwait(false);
                    return;
                }

                if (request is null)
                {
                    var errResponse = new Response { Ok = false, Error = "Null request" };
                    await writer.WriteLineAsync(JsonSerializer.Serialize(errResponse, JsonOptions.Default)).ConfigureAwait(false);
                    return;
                }

                var response = await handler(request).ConfigureAwait(false);
                await writer.WriteLineAsync(JsonSerializer.Serialize(response, JsonOptions.Default)).ConfigureAwait(false);
            }
            catch (OperationCanceledException) { /* shutting down */ }
            catch { /* connection error — discard */ }
        }
    }

    public ValueTask DisposeAsync() => ValueTask.CompletedTask;
}

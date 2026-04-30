using System.Net;
using System.Text;
using System.Text.Json;

namespace KeepScreenAwake.Core.Transport;

public sealed class HttpServer : ITransportServer
{
    private readonly int _port;
    private HttpListener? _listener;

    public HttpServer(int port)
    {
        if (OperatingSystem.IsWindows())
            throw new PlatformNotSupportedException("HttpServer is intended for non-Windows platforms. Use NamedPipeServer on Windows.");
        _port = port;
    }

    public async Task ServeAsync(Func<Request, Task<Response>> handler, CancellationToken ct)
    {
        _listener = new HttpListener();
        _listener.Prefixes.Add($"http://127.0.0.1:{_port}/");
        _listener.Start();

        ct.Register(() =>
        {
            try { _listener.Stop(); } catch { /* ignore */ }
        });

        while (!ct.IsCancellationRequested)
        {
            HttpListenerContext context;
            try
            {
                context = await _listener.GetContextAsync().ConfigureAwait(false);
            }
            catch (HttpListenerException) when (ct.IsCancellationRequested)
            {
                break;
            }
            catch (ObjectDisposedException)
            {
                break;
            }

            _ = Task.Run(() => HandleContextAsync(context, handler, ct), ct);
        }
    }

    private static async Task HandleContextAsync(
        HttpListenerContext context,
        Func<Request, Task<Response>> handler,
        CancellationToken ct)
    {
        var req = context.Request;
        var resp = context.Response;
        resp.ContentType = "application/json";

        try
        {
            Request? request = null;

            if (req.HttpMethod == "GET" && req.Url?.AbsolutePath == "/status")
            {
                request = new Request { Command = CommandType.Status };
            }
            else if (req.HttpMethod == "POST" && req.Url?.AbsolutePath == "/command")
            {
                using var bodyReader = new StreamReader(req.InputStream, req.ContentEncoding);
                var body = await bodyReader.ReadToEndAsync(ct).ConfigureAwait(false);
                request = JsonSerializer.Deserialize<Request>(body, JsonOptions.Default);
            }
            else
            {
                resp.StatusCode = 404;
                var notFound = Encoding.UTF8.GetBytes("{\"ok\":false,\"error\":\"Not found\"}");
                resp.ContentLength64 = notFound.Length;
                await resp.OutputStream.WriteAsync(notFound, ct).ConfigureAwait(false);
                resp.Close();
                return;
            }

            if (request is null)
            {
                resp.StatusCode = 400;
                var badReq = Encoding.UTF8.GetBytes("{\"ok\":false,\"error\":\"Invalid request\"}");
                resp.ContentLength64 = badReq.Length;
                await resp.OutputStream.WriteAsync(badReq, ct).ConfigureAwait(false);
                resp.Close();
                return;
            }

            var response = await handler(request).ConfigureAwait(false);
            var bytes = Encoding.UTF8.GetBytes(JsonSerializer.Serialize(response, JsonOptions.Default));
            resp.StatusCode = response.Ok ? 200 : 500;
            resp.ContentLength64 = bytes.Length;
            await resp.OutputStream.WriteAsync(bytes, ct).ConfigureAwait(false);
        }
        catch (Exception ex)
        {
            try
            {
                resp.StatusCode = 500;
                var err = Encoding.UTF8.GetBytes(JsonSerializer.Serialize(
                    new Response { Ok = false, Error = ex.Message }, JsonOptions.Default));
                resp.ContentLength64 = err.Length;
                await resp.OutputStream.WriteAsync(err, ct).ConfigureAwait(false);
            }
            catch { /* ignore secondary failure */ }
        }
        finally
        {
            resp.Close();
        }
    }

    public ValueTask DisposeAsync()
    {
        _listener?.Stop();
        _listener?.Close();
        return ValueTask.CompletedTask;
    }
}

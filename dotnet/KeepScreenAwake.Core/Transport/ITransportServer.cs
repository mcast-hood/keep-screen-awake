namespace KeepScreenAwake.Core.Transport;

public interface ITransportServer : IAsyncDisposable
{
    Task ServeAsync(Func<Request, Task<Response>> handler, CancellationToken ct);
}

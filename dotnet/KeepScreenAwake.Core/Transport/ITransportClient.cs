namespace KeepScreenAwake.Core.Transport;

public interface ITransportClient : IDisposable
{
    Task<Response> SendAsync(Request request);
}

using System.Text;
using System.Text.Json;

namespace KeepScreenAwake.Core.Transport;

public sealed class HttpTransportClient : ITransportClient
{
    private readonly HttpClient _httpClient;
    private readonly string _baseUrl;
    private bool _disposed;

    public HttpTransportClient(int port)
    {
        _baseUrl = $"http://127.0.0.1:{port}";
        _httpClient = new HttpClient
        {
            Timeout = TimeSpan.FromSeconds(30)
        };
    }

    public async Task<Response> SendAsync(Request request)
    {
        ObjectDisposedException.ThrowIf(_disposed, this);

        var json = JsonSerializer.Serialize(request, JsonOptions.Default);
        var content = new StringContent(json, Encoding.UTF8, "application/json");

        HttpResponseMessage httpResponse;
        try
        {
            httpResponse = await _httpClient.PostAsync($"{_baseUrl}/command", content).ConfigureAwait(false);
        }
        catch (Exception ex)
        {
            return new Response { Ok = false, Error = $"Connection failed: {ex.Message}" };
        }

        var responseBody = await httpResponse.Content.ReadAsStringAsync().ConfigureAwait(false);
        return JsonSerializer.Deserialize<Response>(responseBody, JsonOptions.Default)
               ?? new Response { Ok = false, Error = "Invalid response" };
    }

    public void Dispose()
    {
        if (_disposed) return;
        _disposed = true;
        _httpClient.Dispose();
    }
}

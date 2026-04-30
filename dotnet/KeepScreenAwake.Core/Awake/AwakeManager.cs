using System.Diagnostics;
using System.Runtime.InteropServices;
using System.Runtime.Versioning;

namespace KeepScreenAwake.Core.Awake;

public sealed class AwakeManager : IAwakeManager
{
    private const uint ES_CONTINUOUS = 0x80000000u;
    private const uint ES_SYSTEM_REQUIRED = 0x00000001u;
    private const uint ES_DISPLAY_REQUIRED = 0x00000002u;

    private readonly bool _displayOnly;
    private volatile bool _isActive;
    private bool _disposed;

    // Windows fields
    private Timer? _windowsTimer;

    // macOS fields
    private Process? _caffeinateProcess;

    public bool IsActive => _isActive;

    public AwakeManager(bool displayOnly = false)
    {
        _displayOnly = displayOnly;

        if (!OperatingSystem.IsWindows() && !OperatingSystem.IsMacOS())
            throw new PlatformNotSupportedException("AwakeManager supports Windows and macOS only.");
    }

    public void Enable()
    {
        if (_disposed) throw new ObjectDisposedException(nameof(AwakeManager));

        if (OperatingSystem.IsWindows())
            EnableWindows();
        else if (OperatingSystem.IsMacOS())
            EnableMacOS();

        _isActive = true;
    }

    public void Disable()
    {
        if (_disposed) throw new ObjectDisposedException(nameof(AwakeManager));

        if (OperatingSystem.IsWindows())
            DisableWindows();
        else if (OperatingSystem.IsMacOS())
            DisableMacOS();

        _isActive = false;
    }

    [SupportedOSPlatform("windows")]
    private void EnableWindows()
    {
        AssertWindows();

        _windowsTimer?.Dispose();
        _windowsTimer = new Timer(_ => AssertWindows(), null, TimeSpan.Zero, TimeSpan.FromSeconds(30));
    }

    [SupportedOSPlatform("windows")]
    private void AssertWindows()
    {
        uint flags = ES_CONTINUOUS | ES_DISPLAY_REQUIRED;
        if (!_displayOnly)
            flags |= ES_SYSTEM_REQUIRED;

        SetThreadExecutionState(flags);
    }

    [SupportedOSPlatform("windows")]
    private void DisableWindows()
    {
        _windowsTimer?.Dispose();
        _windowsTimer = null;
        SetThreadExecutionState(ES_CONTINUOUS);
    }

    [SupportedOSPlatform("macos")]
    private void EnableMacOS()
    {
        DisableMacOS();

        var args = _displayOnly ? "-d" : "-d -i";
        _caffeinateProcess = new Process
        {
            StartInfo = new ProcessStartInfo
            {
                FileName = "caffeinate",
                Arguments = args,
                UseShellExecute = false,
                RedirectStandardOutput = true,
                RedirectStandardError = true,
                CreateNoWindow = true
            }
        };
        _caffeinateProcess.Start();
    }

    [SupportedOSPlatform("macos")]
    private void DisableMacOS()
    {
        if (_caffeinateProcess is { HasExited: false })
        {
            try
            {
                _caffeinateProcess.Kill();
                _caffeinateProcess.WaitForExit(2000);
            }
            catch { /* best-effort */ }
        }
        _caffeinateProcess?.Dispose();
        _caffeinateProcess = null;
    }

    public void Dispose()
    {
        if (_disposed) return;
        _disposed = true;

        try { Disable(); } catch { /* best-effort */ }

        if (OperatingSystem.IsWindows())
        {
            _windowsTimer?.Dispose();
        }
        else if (OperatingSystem.IsMacOS())
        {
            _caffeinateProcess?.Dispose();
        }
    }

    [SupportedOSPlatform("windows")]
    [DllImport("kernel32.dll", SetLastError = true)]
    private static extern uint SetThreadExecutionState(uint esFlags);
}

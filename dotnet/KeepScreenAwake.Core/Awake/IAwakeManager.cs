namespace KeepScreenAwake.Core.Awake;

public interface IAwakeManager : IDisposable
{
    void Enable();
    void Disable();
    bool IsActive { get; }
}

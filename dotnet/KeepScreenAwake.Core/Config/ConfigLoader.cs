using YamlDotNet.Serialization;
using YamlDotNet.Serialization.NamingConventions;

namespace KeepScreenAwake.Core.Config;

public static class ConfigLoader
{
    private static readonly string[] ValidModes = { "always", "toggle", "schedule" };

    public static AppConfig LoadOrDefault(string path)
    {
        if (!File.Exists(path))
            return new AppConfig();

        var yaml = File.ReadAllText(path);
        var deserializer = new DeserializerBuilder()
            .WithNamingConvention(CamelCaseNamingConvention.Instance)
            .IgnoreUnmatchedProperties()
            .Build();

        var config = deserializer.Deserialize<AppConfig>(yaml) ?? new AppConfig();
        Validate(config);
        return config;
    }

    private static void Validate(AppConfig config)
    {
        if (!ValidModes.Contains(config.Mode, StringComparer.OrdinalIgnoreCase))
            throw new InvalidOperationException(
                $"Invalid mode '{config.Mode}'. Must be one of: {string.Join(", ", ValidModes)}");

        config.Mode = config.Mode.ToLowerInvariant();
    }
}

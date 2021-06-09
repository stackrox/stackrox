type Config = {
    backgroundColor: string;
    color: string;
    enabled: boolean;
    size: string;
    text: string;
};

export type SystemConfig = {
    publicConfig: {
        footer: Config;
        header: Config;
        loginNotice?: {
            enabled: boolean;
            text: string;
        };
    } | null;
};

export type TelemetryConfig = {
    enabled: boolean;
    lastSetTime: string;
};

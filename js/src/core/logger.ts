enum LoggerKey {
    RESPONSE = "RESPONSE",
    INFO = "INFO",
    ERROR = "ERROR"
}

type Logger = (key: LoggerKey, message: string) => void;

export { LoggerKey }
export type { Logger }
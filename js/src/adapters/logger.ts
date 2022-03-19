import { Logger, LoggerKey } from "../core/logger"

const getLogger = (): Logger => {
    return (key: LoggerKey, message: string) => {
        if (key === LoggerKey.INFO) console.log(message)
        if (key === LoggerKey.ERROR) console.error(message)
        if (key === LoggerKey.RESPONSE) console.log("Not showing full response [getLogger]")
    };
}

export {
    getLogger
}
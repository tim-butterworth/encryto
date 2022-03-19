import { BehaviorSubject, Subject, Subscription } from "rxjs";
import { Logger, LoggerKey } from "../logger";
import { connection, ConnectionState, IncomingMessage, SocketWrapper } from "../socket";
import { HandshakeMachine, Transition, State } from "./stateMachine";

type Decryption = {
    getPem: () => string
    decrypt: (message: number[]) => Promise<string>
}

type Encryption = {
    encrypt: (message: string) => Promise<number[]>
}
type EncryptionProvider = (pem: number[]) => Promise<Encryption>

type HandshakeWorkflow = (
    socketWrapper: SocketWrapper,
    { success, failure }: {
        success: (socketWrapper: SocketWrapper) => void,
        failure: () => void,
    }
) => void;

type SubscribeOf<T> = (f: (state: T) => void) => void; 
type SocketSubscriber = {
    connectionState: SubscribeOf<ConnectionState>;
    incoming: SubscribeOf<IncomingMessage>;
};
const keysOf = <T>(obj: T): Array<keyof T> => {
    return Object.keys(obj) as Array<keyof T>;
} 
const getSocketSubscriber = (socket: SocketWrapper): SocketSubscriber & { unsubscribeAll: () => void; } => {
    const subscriptions: {
        [key in keyof SocketSubscriber]?: Subscription 
    } = {};

    return {
        connectionState: (sub: (state: ConnectionState) => void) => {
            const connectionSub = socket.connectionState.subscribe(sub);
            subscriptions["connectionState"] = connectionSub;
        },
        incoming: (sub: (incoming: IncomingMessage) => void) => {
            const incomingSubscription = socket.incoming.subscribe(sub);
            subscriptions["incoming"] = incomingSubscription;
        },
        unsubscribeAll: () => {
            keysOf(subscriptions).forEach((key) => {
                const subscription = subscriptions[key];
                if (subscription) {
                    subscription.unsubscribe();
                    subscriptions[key] = undefined;
                }
            })
        }
    }
}

type MessageVariant = {
    variant: string;
}
const isVariant = (obj: object): obj is MessageVariant => "variant" in obj;
const isServerKeyMessage = (obj: MessageVariant): obj is MessageVariant & {
    body: {
        publicKey: number[];
    }
} => obj.variant === "ServerKey";
const isVerificationMessage = (obj: MessageVariant): obj is MessageVariant & {
    body: {
        message: number[];
    }
} => obj.variant === "Verification";
const isRegularMessage = (obj: MessageVariant): obj is MessageVariant & {
    body: number[]
} => obj.variant === "Message";

type HandshakeMachineWorkflowContext = {
    serverEncryption?: Encryption;
    verificationCode?: string;
    message?: string;
}
const getHandshakeWorkflow = (
    logger: Logger,
    decryption: Decryption,
    encryptionProvider: EncryptionProvider,
    handshakeMachineProvider: () => HandshakeMachine<HandshakeMachineWorkflowContext>,
): HandshakeWorkflow => (
    socketWrapper: SocketWrapper,
    { success, failure }: {
        success: (socketWrapper: SocketWrapper) => void,
        failure: () => void,
    }
) => {
    const handshakeMachine = handshakeMachineProvider();

    const socketSubscriber = getSocketSubscriber(socketWrapper);
    socketSubscriber.connectionState((state: ConnectionState) => {
        logger(LoggerKey.INFO, `"CONNECTION STATE -> ", ${state}`);
        if (connection.isConnected(state)) {
            logger(LoggerKey.INFO, "socket is connected");

            handshakeMachine.pureTransition(Transition.CONNECT);
        }

        if (connection.isDisconnected(state)) {
            handshakeMachine.pureTransition(Transition.ERROR);
        }

        if (connection.isPending(state)) {
            logger(LoggerKey.INFO, "is pending");
        }
    });
    socketSubscriber.incoming((json: IncomingMessage) => {
        if (isVariant(json)) {
            const variant = json.variant;
            if (isServerKeyMessage(json)) {
                logger(LoggerKey.RESPONSE, JSON.stringify(json, null, 2))

                encryptionProvider(json.body.publicKey)
                    .then((serverEncryption) => {
                        handshakeMachine.transition(Transition.SERVER_KEY_RECEIVED, (c: {}) => ({
                            ...c,
                            serverEncryption
                        }))
                    })
                    .catch((error) => {
                        logger(LoggerKey.ERROR, `${error}`)
                    })
            }
      
            if (variant === "KeyReceived") {
                handshakeMachine.pureTransition(Transition.CONFIRMED_KEY);
            }
      
            if (isVerificationMessage(json)) {
                logger(LoggerKey.RESPONSE, JSON.stringify(json, null, 2))
                decryption.decrypt(json.body.message)
                    .then((verificationCode: string) => {
                        handshakeMachine.transition(Transition.RECEIVED_VERIFICATION, (context: HandshakeMachineWorkflowContext) => ({
                            ...context,
                            verificationCode
                        }))
                    })
                    .catch((e) => {
                        logger(LoggerKey.ERROR, `"Error decrypting" ${e}`);
                        handshakeMachine.pureTransition(Transition.ERROR)
                    })
            }
      
            if (isRegularMessage(json)) {
                decryption.decrypt(json.body)
                    .then((decrypted: string) => {
                        handshakeMachine.transition(Transition.CONFIRM_VERIFICATION, (context: HandshakeMachineWorkflowContext) => ({
                            ...context,
                            message: decrypted
                        }))
                    })
                    .catch((e) => logger(LoggerKey.ERROR, `${e}`))
            }
          } else {
            logger(LoggerKey.ERROR, `"Invalid response from the server", ${JSON.stringify(json, null, 2)}`);
          }
    });

    const machineSubscription = handshakeMachine.subscribe(({ state, context }) => {
        logger(LoggerKey.INFO, `HANDSHAKE_MACHINE: ${state}`);
        if (state === State.ERROR) {
            logger(LoggerKey.ERROR, "ERROR")

            socketSubscriber.unsubscribeAll();
            machineSubscription.unsubscribe();

            failure();
        }

        if (state === State.CONNECTED) {
            socketWrapper.dispatch.next({
                varient: "SetPublicKey",
                Data: {
                    PublicKey: decryption.getPem()
                }
            })

            handshakeMachine.pureTransition(Transition.SENT_KEY)
        }

        if (state === State.CLIENT_KEY_RECEIVED) {
            socketWrapper.dispatch.next({
                varient: "GetPublicKey",
            })

            handshakeMachine.pureTransition(Transition.REQUESTED_SERVER_KEY)
        }

        if (state === State.KEYS_EXCHANGED) {
            logger(LoggerKey.INFO, `${context["serverEncryption"]}`);
            socketWrapper.dispatch.next({
                varient: "GetVerification"
            })

            handshakeMachine.pureTransition(Transition.REQUEST_VERIFICATION);
        }

        if (state === State.SERVER_VERIFICATION_RECEIVED) {
            logger(LoggerKey.INFO, `"VERIFICATION_RECEIVED", ${context["verificationCode"]}`)
            logger(LoggerKey.INFO, `CONTEXT KEYS: [${Object.keys(context)}]`)
            if (context.serverEncryption && context.verificationCode) {
                const serverEncryption = context["serverEncryption"] as Encryption;

                serverEncryption.encrypt(context["verificationCode"]).then((encrypted: number[]) => {
                    socketWrapper.dispatch.next({
                        varient: "Verify",
                        Data: {
                            message: encrypted,
                        }
                    })
                });

                handshakeMachine.pureTransition(Transition.SENT_VERIFICATION)
            } else {
                logger(LoggerKey.ERROR, "missing encryption or the verification code")
                failure()
            }
        }

        if (state === State.VERIFICATION_CONFIRMED) {
            if (context.message) {
                logger(LoggerKey.INFO, `DECRYPTED: ${context.message}`)
            } else {
                logger(LoggerKey.INFO, "Message is missing.....")
            }
            socketSubscriber.unsubscribeAll();

            if (context.message && context.serverEncryption) {
                success(encryptedSocket({ 
                    underlyingSocket: socketWrapper, 
                    serverEncryption: context.serverEncryption, 
                    clientDecryption: decryption,
                    initialMessage: context.message
                }));
            }
        }
    })
};

const encryptedSocket = ({
    underlyingSocket,
    serverEncryption,
    clientDecryption,
    initialMessage,
}: {
    underlyingSocket: SocketWrapper,
    serverEncryption: Encryption,
    clientDecryption: Decryption,
    initialMessage: string,
}): SocketWrapper => {
    const encryptedDispatch = new Subject<{}>();
    encryptedDispatch.subscribe((message: object) => {
        const messageString = JSON.stringify(message);
        serverEncryption
            .encrypt(messageString)
            .then((encrypted) => {
                console.log("sending encrypted message")
                console.log(encrypted)
                underlyingSocket.dispatch.next({
                    varient: "Message",
                    Data: encrypted
                });
            })
            .catch(console.error)
    });

    const decryptedIncoming = new BehaviorSubject<{}>(initialMessage);
    underlyingSocket.incoming.subscribe((json: object) => {
        if (isVariant(json)) {
            if (isRegularMessage(json)) {
                clientDecryption.decrypt(json.body)
                    .then((decrypted) => {
                        decryptedIncoming.next(decrypted)
                    })
                    .catch(console.error)
            }
        }
    })

    return {
        connectionState: underlyingSocket.connectionState,
        dispatch: encryptedDispatch,
        incoming: decryptedIncoming,
    }
}

export {
    getHandshakeWorkflow
}
export type {
    HandshakeWorkflow, Decryption, Encryption, EncryptionProvider
}

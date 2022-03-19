import { BehaviorSubject, Observable } from "rxjs";
import { HandshakeWorkflow } from "./handshake/workflow";
import { MachineProvider, StateMap } from "./machineHelpers";
import { connection, ConnectionState, IncomingMessage, SocketWrapper } from "./socket";

enum State {
    INITIAL = "INITIAL",
    NEW_SOCKET = "NEW_SOCKET",
    HANDSHAKE = "HANDSHAKE",
    CONNECTED = "CONNECTED"
}
enum InitialTransitions {
    BEGIN = "BEGIN",
}
enum CreateSocketTransitions {
    ERROR = "(CREATE_SOCKET) ERROR",
    SUCCESS = "(CREATE_SOCKET) SUCCESS",
}
enum HandshakeTransitions {
    ERROR = "(HANDSHAKE) ERROR",
    SUCCESS = "(HANDSHAKE) SUCCESS"
}
enum ConnectedTransitions {
    ERROR = "(CONNECTED) ERROR"
}

type StateTransitionMap = {
    [State.INITIAL]: InitialTransitions,
    [State.NEW_SOCKET]: CreateSocketTransitions,
    [State.HANDSHAKE]: HandshakeTransitions,
    [State.CONNECTED]: ConnectedTransitions,
};

const stateMap: StateMap<StateTransitionMap> = {
    [State.INITIAL]: {
        transitions: {
            "BEGIN": State.NEW_SOCKET
        },
    },
    [State.NEW_SOCKET]: {
        transitions: {
            "(CREATE_SOCKET) ERROR": State.INITIAL,
            "(CREATE_SOCKET) SUCCESS": State.HANDSHAKE
        },
    },
    [State.HANDSHAKE]: {
        transitions: {
            "(HANDSHAKE) ERROR": State.INITIAL,
            "(HANDSHAKE) SUCCESS": State.CONNECTED
        }
    },
    [State.CONNECTED]: {
        transitions: {
            "(CONNECTED) ERROR": State.INITIAL,
        }
    }
}

const hasEncryptedSocket = (obj: object): obj is { encryptedSocket: SocketWrapper } => {
    return "encryptedSocket" in obj;
}
const hasSocket = (obj: object): obj is { "socket": SocketWrapper } => {
    return "socket" in obj;
}

const getConnectionMachine = (
    socketFactory: () => SocketWrapper, 
    handshakeWorkflow: HandshakeWorkflow,
    machineProvider: MachineProvider
): {
    incoming: Observable<object>,
    dispatch: (message: object) => void,
} => {
    const incoming = new BehaviorSubject<object>({});
    const outgoing = new BehaviorSubject<object>({});
    const machine = machineProvider({ map: stateMap, initialState: State.INITIAL});

    const subscription = machine.subscribe(({ state, context }) => {
        if (state === State.INITIAL) {
            machine.pureTransition(InitialTransitions.BEGIN);
        }

        if (state === State.NEW_SOCKET) {
            try {
                const socketWrapper = socketFactory();
                machine.transition(CreateSocketTransitions.SUCCESS, (context: object) => ({
                    ...context,
                    socket: socketWrapper,
                }))
            } catch (error) {                
                console.log("Error creating socket connection", error);
                machine.pureTransition(CreateSocketTransitions.ERROR)
            }
        }

        if (state === State.HANDSHAKE) {
            if (hasSocket(context)) {
                const socket = context.socket;
                handshakeWorkflow(socket, {
                    success: (encryptedSocket: SocketWrapper) => {
                        machine.transition(HandshakeTransitions.SUCCESS, (context: object) => ({
                            ...context,
                            encryptedSocket,
                        }));
                    },
                    failure: () => {
                        console.log("Error during handshake");
                        machine.pureTransition(HandshakeTransitions.ERROR);                    
                    },
                });
            } else {
                console.log("Error, socket is not in the context");
                machine.pureTransition(HandshakeTransitions.ERROR);
            }
        }

        if (state === State.CONNECTED) {
            if (hasEncryptedSocket(context)) {
                const socket = context.encryptedSocket;

                const incomingSubscription = socket.incoming.subscribe((incomingMessage: IncomingMessage) => {
                    console.log("IN [CONNECTION]", incomingMessage)

                    const parsed = JSON.parse(`${incomingMessage}`);
                    if ("variant" in parsed) {
                        if (parsed["variant"] === "AvailableActions") {
                            socket.dispatch.next({
                                Varient: "connect"
                            })
                            outgoing.subscribe((m) => {
                                console.log("Sending message", JSON.stringify(m, null, 2));
                                socket.dispatch.next(m);
                            });
                        }

                        if (parsed["variant"] === "Message") {
                            incoming.next(parsed);
                        }
                    }
                })

                const socketSubscription = socket.connectionState.subscribe((connectionState: ConnectionState) => {
                    if (connection.isDisconnected(connectionState)) {
                        socketSubscription.unsubscribe();
                        incomingSubscription.unsubscribe();
                        setTimeout(
                            () => {
                                console.log("ERROR with the connection, going to re-connect in ~1second")
                                machine.pureTransition(ConnectedTransitions.ERROR)
                            },
                            1000
                        );
                    }
                });
            }
        }
    });
    console.log(subscription);

    return {
        incoming: incoming.asObservable(),
        dispatch: (message: object) => outgoing.next(message),
    }
}

export {getConnectionMachine};
